package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/acityinohio/baduk"
	"github.com/acityinohio/blockcy"
)

type Gob struct {
	multi        string
	serverPub    string
	serverPriv   string
	blackPK      string
	whitePK      string
	blackWinAddr string
	whiteWinAddr string
	blackMove    bool
	state        baduk.Board
}

var templates = template.Must(template.ParseGlob("templates/*"))

//Keeping it all in memory
var boards []Gob

func init() {
	blockcy.Config.Coin, blockcy.Config.Chain = "bcy", "test"
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/games/new", newGameHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "index.html", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func newGameHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/games/new", http.StatusFound)
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
	board.blackWinAddr = f("blackWinAddr")
	board.whiteWinAddr = f("whiteWinAddr")
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
	for i, v := range wip.ToSign {
		wip.Signatures[i], err = signTX(pair.Private, v)
		wip.PubKeys[i] = pair.Public
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	//Send transaction
	tx, err := blockcy.SendTX(wip)
	fmt.Fprintf(w, "%+v\n", tx)
	return
}
