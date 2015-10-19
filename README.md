# Bitduck

Building an ephemeral Go (Baduk) webapp, using Go, with a library I built with Go, integrated with [BlockCypher's API/Testnet](http://blockcypher.com/). Right now it's in a deep alpha state, but it works! Allows 2-of-2 multsig games, where the moves are embedded into BlockCypher's blockchain...can be easily forked to work with bitcoin through BlockCypher's API.

# To Install

You must have Go installed. Clone into the repository, then run `go build` in your directory. It will make an executable that will run as the web server. Right now, there is no permanent memory state; games are destroyed as soon as the executable quits. However, they can be reconstructed via blockchain OP_returns, just by using the http://yourhost.com/games/MultiSigAddress?blackpk=BLACKPUBKEY&whitepk=WHITEPUBKEY. Pretty cool huh?

# To Do

* Flesh out this readme
* Score/reach endgame winning player
* General code cleanup
