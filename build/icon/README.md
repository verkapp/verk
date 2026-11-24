# Icône de Verk

Le monogramme « v. » : un v minuscule et son point, repris du nom « verk. » de la barre latérale.
Les SVG de ce dossier sont les sources ; les PNG, `.icns` et `.ico` en sont dérivés.

| Source | Fichier produit | Taille |
| --- | --- | --- |
| `appicon.svg` | `build/appicon.png` | 1024 × 1024 |
| `appicon.svg` | `src/trayicons/appicon.png` (icône de fenêtre) | 256 × 256 |
| `tray-color.svg` | `src/trayicons/color.png` (zone de notification Windows, Linux) | 64 × 64 |
| `tray-template.svg` | `src/trayicons/template.png` (barre des menus macOS) | 44 × 44 |

Le favicon (`frontend/public/favicon.svg`) et le logo de la barre latérale reprennent le même tracé
sur la grille 64 × 64.

Après une modification, exporter les PNG avec fond transparent (par exemple avec Chrome headless :
`chrome --headless=new --default-background-color=00000000 --window-size=1024,1024 --screenshot=build/appicon.png build/icon/appicon.svg`),
puis régénérer les icônes macOS et Windows :

```bash
wails3 task common:generate:icons
```
