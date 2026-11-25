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

- Des scripts d’installation : `scripts/install.sh` pour macOS (télécharge la dernière release) et
  Linux (compile depuis les sources), `scripts/install.ps1` pour Windows (installeur silencieux).
  Les deux vérifient l’empreinte SHA-256 et passent par `gh` ou un jeton, le dépôt étant privé.

### Corrigé

- Sous Windows, coller du texte avec Ctrl+V l’insérait deux fois, et Ctrl+Z annulait deux étapes :
  les raccourcis d’édition sont laissés à la vue web.
- Le bouton « Marquer faite » du panneau de tâche était coupé : « Supprimer » et « Ne pas faire »
  deviennent des icônes, et les boutons passent à la ligne quand la place manque.

## [3.0.0] - 2026-11-24

Le dépôt de synchronisation change de format : mets à jour tous tes appareils. Une version 2.0
refuse de synchroniser avec un dépôt migré, sans rien abîmer.

### Modifié

- Le dépôt de synchronisation range un fichier par tâche, dans un dossier par profil
  (`profiles/<profil>/tasks/<tâche>.json`), au lieu d’un seul `todo.json`. Les commits ne touchent que
  les tâches modifiées et l’historique d’une tâche se lit avec `git log`. Le dépôt est migré à la
  première synchronisation. Les appareils restés en 2.0 refusent ensuite de synchroniser et
  demandent une mise à jour : mets à jour tous tes appareils.
- Sous Windows, la barre de menus Fichier, Édition, Affichage est masquée : la fenêtre commence
  directement sous la barre de titre. Les raccourcis clavier (Ctrl+N, Ctrl+1 à 4, Ctrl+,) restent actifs.

## [2.0.0] - 2026-11-24

Todo devient **Verk**. Les tâches, pièces jointes et réglages sont repris automatiquement au premier
lancement.

### Modifié

- L’application s’appelle désormais **Verk**, avec une nouvelle icône : le monogramme « v. ». Au premier
  lancement, les données de Todo sont reprises (dossier `todo` → `verk`, `todo.db` → `verk.db`). La
  synchronisation git reste compatible avec les appareils encore sous Todo. Sur macOS, l’autorisation des
  notifications peut être redemandée.
- Les réglages sont rangés en onglets : Général, Apparence, Profils, Claude, Synchronisation et
  Application. Le dernier onglet ouvert est retenu, et une pastille signale une section à vérifier
  (synchronisation en erreur, mise à jour prête, Claude Code introuvable).
- Les boutons et textes d’accent sont plus contrastés : le texte des boutons atteint au moins 4,5:1 avec
  chacun des six accents, en clair comme en sombre.

### Ajouté

- Un thème sombre (Réglages → Apparence) : clair, sombre, ou celui du système, qui change avec lui. Sous
  Windows, la barre de titre, la bordure et la barre de menus suivent immédiatement ; sur macOS, les menus
  système suivent au prochain lancement.

### Corrigé

- Sous Windows, la barre de menus (Fichier, Édition, Affichage) et ses raccourcis clavier manquaient depuis
  le passage à Wails v3. Le menu Édition est aussi traduit.

## [1.4.0] - 2026-11-24

### Ajouté

- Report automatique, réglable par profil (Réglages → Profils) : à l’heure choisie, les tâches du jour
  non terminées et les tâches en retard passent à demain (ou lundi), comme avec « Reprendre ». Une tâche
  planifiée plus tard dans la soirée reste en place ; si l’app était fermée, le report se fait au lancement.
- Un installeur Windows (`Todo-windows-x64-setup.exe`) : installation pour l’utilisateur, sans droits
  administrateur, avec raccourcis dans le menu Démarrer et sur le bureau.
- Marquer une tâche comme « pas à faire » (bouton ⊘ sur la carte ou « Ne pas faire » dans le panneau) :
  elle sort des listes, des retards et des rappels sans compter comme terminée. Elle reste visible dans
  l’historique, en gris, et se rouvre comme une tâche terminée.
- Un réglage pour désactiver entièrement l’intégration Claude : plus aucun appel à Claude Code, le coach et le
  découpage de tâches disparaissent de l’interface.
- Un bouton « Rafraîchir » à côté de « Nouvelle tâche » pour recharger la liste à la demande.
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
