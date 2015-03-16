# Bitduck

Building an ephemeral Go (Baduk) webapp, using Go, with a library I built with Go, integrated with [BlockCypher's API/Testnet](http://blockcypher.com/). Right now it's in a deep alpha state, but it works! It simulates building a new multisig transaction on BlockCypher's testnet, then allows your to prompt the users to sign their moves using their own private keys.

# To Install

You must have Go installed. Clone into the repository, then run `go build` in your directory. It will make an executable that will run as the web server. Right now, there is no permanent memory state---games are destroyed as soon as the executable quits.

# To Do

* Flesh out this readme
* Add some persistent storage
* Score/reach endgame winning player
* Automatically sign multisig in winning player's favor
* General code cleanup
