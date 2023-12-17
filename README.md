# dispel-multi

This repository is a monorepo that contains multiple projects that helps to play a Dispel Multiplayer game.

* **console** - A main server handling the state preservation and communication with the clients.
* **backend** - A backend server that handles the commands send by the `DispelMulti.exe`. It exchanges the data with the **console**.
* **launcher** - A launcher for the game. It is a GUI app that is responsible for resolving all problems with interconnecting the game, the **console** and the **backend**.
