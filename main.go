package main

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/acityinohio/baduk"
	"github.com/acityinohio/blockcy"
)

type Gob struct {
	multi      string
	serverPub  string
	serverPriv string
	blackPK    string
	whitePK    string
	blackMove  bool
	state      baduk.Board
}

var templates = template.Must(template.ParseGlob("templates/*"))

//Keeping it all in memory
var boards map[string]Gob

func init() {
	boards = make(map[string]Gob)
	blockcy.Config.Coin, blockcy.Config.Chain = "bcy", "test"
	blockcy.Config.Token = "e212e91ac4d218cbc18f7eb3975122e3"
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/games/", gameHandler)
	http.HandleFunc("/games/new/", newGameHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
	multi := r.URL.Path[len("/games/"):]
	board, ok := boards[multi]
	if !ok {
		http.Error(w, "Board does not exist", http.StatusInternalServerError)
		return
	}
	if r.Method == "POST" {
		moveHandler(w, r, board)
		return
	}
	type gameTemp struct {
		Multi     string
		PrettySVG string
		BlackMove bool
	}
	necessary := gameTemp{board.multi, board.state.PrettySVG(), board.blackMove}
	err := templates.ExecuteTemplate(w, "game.html", necessary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func moveHandler(w http.ResponseWriter, r *http.Request, board Gob) {
	//Get move and signature
	//Verify move/signature
	//If verified, set board, update memory
	//Otherwise, revert back to board setting
	return
}

func newGameHandler(w http.ResponseWriter, r *http.Request) {
	f := r.FormValue
	var board Gob
	var err error
	//Initialize Board
	sz, err := strconv.Atoi(f("size"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.state.Init(sz)
	board.blackPK = f("blackPK")
	board.whitePK = f("whitePK")
	board.blackMove = true
	//Generate pub/priv key for this board
	pair, err := blockcy.GenAddrPair()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.serverPub = pair.Public
	board.serverPriv = pair.Private
	//Simulate funding multisig wallet from server
	pair.Faucet(5e6)
	txskel, err := blockcy.SkelMultiTX(pair.Address, "", 4e6, false, 2, []string{board.serverPub, board.blackPK, board.whitePK})
	wip, err := blockcy.NewTX(txskel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Sign transaction
	wip.Signatures = make([]string, len(wip.ToSign))
	wip.PubKeys = make([]string, len(wip.ToSign))
	for i, v := range wip.ToSign {
		wip.Signatures[i], err = signTX(pair.Private, v)
		wip.PubKeys[i] = pair.Public
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	//Send transaction
	//tx, err := blockcy.SendTX(wip)
	//Just use WIP for now
	board.multi = wip.Trans.Outputs[0].Addresses[0]
	boards[board.multi] = board
	http.Redirect(w, r, "/games/"+board.multi, http.StatusFound)
	return
}
