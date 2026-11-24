# Icône de Verk

Le monogramme « v. » : un v minuscule et son point, repris du nom « verk. » de la barre latérale.
Les SVG de ce dossier sont les sources ; les PNG, `.icns` et `.ico` en sont dérivés.

| Source | Fichier produit | Taille |
| --- | --- | --- |
| `appicon.svg` | `build/appicon.png` | 1024 × 1024 |
| `appicon.svg` | `src/trayicons/appicon.png` (icône de fenêtre) | 256 × 256 |
| `tray-color.svg` | `src/trayicons/color.png` (zone de notification Windows, Linux) | 64 × 64 |
| `tray-template.svg` | `src/trayicons/template.png` (barre des menus macOS) | 44 × 44 |
| `dmg-file-icon.svg` | `build/darwin/dmg-file-icon.png` puis `.icns` (icône du fichier .dmg dans le Finder) | 1024 × 1024 |
| `dmg-background.svg` | `build/darwin/dmg-background.png` (fond de la fenêtre du .dmg) | 540 × 380 |

Le favicon (`frontend/public/favicon.svg`) et le logo de la barre latérale reprennent le même tracé
sur la grille 64 × 64.

Après une modification, exporter les PNG avec fond transparent (par exemple avec Chrome headless :
`chrome --headless=new --default-background-color=00000000 --window-size=1024,1024 --screenshot=build/appicon.png build/icon/appicon.svg`),
puis régénérer les icônes macOS et Windows :

```bash
wails3 task common:generate:icons
```

Pour l'icône du fichier .dmg :

```bash
wails3 generate icons -input build/darwin/dmg-file-icon.png -macfilename build/darwin/dmg-file-icon.icns -windowsfilename /tmp/unused.ico
```

Le fond du .dmg suit la mise en page de `wails3 tool package` : icône de l'app en (151, 164),
lien Applications en (389, 164). Son texte utilise Inter depuis `frontend/node_modules` : lancer
`npm ci` dans `frontend/` avant l'export.
