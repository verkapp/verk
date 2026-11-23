# Journal des versions

Les notes de chaque version vivent ici. Avant de créer un tag `vX.Y.Z`, ajoute une section
`## [X.Y.Z] - AAAA-MM-JJ` : la CI en extrait le contenu pour alimenter la GitHub Release, et
l’application l’affiche dans Réglages → Mises à jour.

Format :

- **Ajouté** pour les nouveautés ;
- **Modifié** pour les changements de comportement ;
- **Corrigé** pour les corrections ;
- **Retiré** pour ce qui disparaît.

## [Non publié]

### Ajouté

- Un installeur Windows (`Todo-windows-x64-setup.exe`) : installation pour l’utilisateur, sans droits
  administrateur, avec raccourcis dans le menu Démarrer et sur le bureau.
- Marquer une tâche comme « pas à faire » (bouton ⊘ sur la carte ou « Ne pas faire » dans le panneau) :
  elle sort des listes, des retards et des rappels sans compter comme terminée. Elle reste visible dans
  l’historique, en gris, et se rouvre comme une tâche terminée.
- Un réglage pour désactiver entièrement l’intégration Claude : plus aucun appel à Claude Code, le coach et le
  découpage de tâches disparaissent de l’interface.
- La liste se rafraîchit dès qu’une synchronisation git se termine, y compris celle du démarrage, sans attendre
  le prochain sondage.
- Une barre de menus native (Fichier, Édition, Affichage) avec raccourcis : nouvelle tâche, rafraîchir, vues et
  réglages. Sur macOS, les menus Todo et Fenêtre habituels sont aussi présents.
- Une icône dans la barre des menus (macOS) ou la zone de notification (Windows, Linux) : ouvrir Todo, créer une
  tâche, réduire la fenêtre dans la barre ou quitter. Sur macOS, un clic sur l’icône du Dock rouvre la fenêtre.
- Une barre de titre intégrée : sur macOS, les boutons de fenêtre se posent sur la barre latérale ; sur
  Windows, la barre de titre et les menus prennent les couleurs de l’application.
- Un réglage « Fermer dans la barre » : fermer la fenêtre la range au lieu de quitter l’application. Il
  s’applique immédiatement.

### Modifié

- L’application passe à Wails v3 : l’icône de tray et les menus utilisent désormais le support natif
  du framework.
- Les notes de version sont rendues avec un vrai moteur Markdown.

## [1.3.0] - 2026-11-05

### Ajouté

- Pièces jointes de tout type, collables avec Ctrl+V (PDF, documents, archives…) et pas seulement
  des images. Les images gardent leur aperçu, les autres fichiers s’ouvrent avec l’application
  système.
- Coche en cascade : terminer toutes les sous-tâches termine la tâche, et terminer une tâche
  coche ses sous-tâches.
- Les notes de version s’affichent dans Réglages → Mises à jour.

### Modifié

- Sections repliables dans les listes et l’historique.
- Choix simplifié du moment de la journée pour les échéances.
- Synchronisation git regroupée et relancée au retour de focus.
- Les notes de release GitHub sont extraites de ce journal.

## [1.2.3] - 2026-11-02

### Corrigé

- Détection de la CLI `gh` quand l’application est lancée depuis le Finder.

## [1.2.2] - 2026-11-02

### Modifié

- Les mises à jour automatiques sont actives après une migration d’ancienne base.

### Corrigé

- Migration des bases créées par les versions précédentes.

## [1.2.1] - 2026-11-02

### Corrigé

- Le profil actif est repris à la création d’une nouvelle tâche.

## [1.2.0] - 2026-11-02

### Ajouté

- Profils pour classer les tâches.
- Réordonnancement manuel des tâches.
- Notifications de bureau.

## [1.1.0] - 2026-11-01

### Ajouté

- Synchronisation git optionnelle.

## [1.0.0] - 2026-11-01

### Ajouté

- Première version publique : tâches, sous-tâches, échéances, images et coach Claude Code.
