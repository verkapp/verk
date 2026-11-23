#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

extern void todoTrayClicked(int id);

// Ids mirror the tray* constants in menu.go.
enum { TodoTrayOpen = 1, TodoTrayNewTask, TodoTrayHide, TodoTrayQuit };

@interface TodoTrayTarget : NSObject
- (void)clicked:(NSMenuItem *)sender;
@end

@implementation TodoTrayTarget
- (void)clicked:(NSMenuItem *)sender {
  todoTrayClicked((int)sender.tag);
}
@end

static NSStatusItem *todoStatusItem = nil;
static TodoTrayTarget *todoTarget = nil;

static void todoAddItem(NSMenu *menu, NSString *title, int tag) {
  NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:@selector(clicked:) keyEquivalent:@""];
  item.target = todoTarget;
  item.tag = tag;
  [menu addItem:item];
  [item release];
}

// Clicking the Dock icon while the window is hidden in the tray reopens it.
// Wails' delegate does not implement this, so it is added at runtime.
static BOOL todoHandleReopen(id self, SEL _cmd, NSApplication *app, BOOL hasVisibleWindows) {
  if (!hasVisibleWindows) {
    todoTrayClicked(TodoTrayOpen);
  }
  return YES;
}

static void todoInstallReopenHandler(void) {
  id delegate = [NSApp delegate];
  if (delegate == nil) {
    return;
  }
  SEL sel = @selector(applicationShouldHandleReopen:hasVisibleWindows:);
  if (![delegate respondsToSelector:sel]) {
    class_addMethod([delegate class], sel, (IMP)todoHandleReopen, "c@:@c");
  }
}

void todoTrayStart(const void *png, int length) {
  NSData *data = [[NSData alloc] initWithBytes:png length:length];
  dispatch_async(dispatch_get_main_queue(), ^{
    if (todoStatusItem != nil) {
      [data release];
      return;
    }
    todoTarget = [TodoTrayTarget new];
    todoStatusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength] retain];

    NSImage *image = [[NSImage alloc] initWithData:data];
    [data release];
    image.size = NSMakeSize(18, 18);
    [image setTemplate:YES];
    todoStatusItem.button.image = image;
    todoStatusItem.button.toolTip = @"Todo";
    [image release];

    NSMenu *menu = [[NSMenu alloc] initWithTitle:@"Todo"];
    menu.autoenablesItems = NO;
    todoAddItem(menu, @"Ouvrir Todo", TodoTrayOpen);
    todoAddItem(menu, @"Nouvelle tâche", TodoTrayNewTask);
    [menu addItem:[NSMenuItem separatorItem]];
    todoAddItem(menu, @"Réduire dans la barre", TodoTrayHide);
    [menu addItem:[NSMenuItem separatorItem]];
    todoAddItem(menu, @"Quitter Todo", TodoTrayQuit);
    todoStatusItem.menu = menu;
    [menu release];

    todoInstallReopenHandler();
  });
}

void todoTrayStop(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    if (todoStatusItem == nil) {
      return;
    }
    [[NSStatusBar systemStatusBar] removeStatusItem:todoStatusItem];
    [todoStatusItem release];
    todoStatusItem = nil;
  });
}

void todoActivateApp(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [NSApp activateIgnoringOtherApps:YES];
  });
}
