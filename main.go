package main

import (
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/acityinohio/baduk"
	"github.com/blockcypher/gobcy"
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
var boards map[string]*Gob
var bcy gobcy.API

func init() {
	boards = make(map[string]*Gob)
	var board Gob
	board.state.Init(4)
	board.blackPK = "024e57e7d387e40add43a71fa998cf9004deb73741b57dcff4cfe31251ceab64ce"
	board.whitePK = "024f20d3f0d97e9cbce9e7d0586b034140b5d320fd86ea4ce14d1d82e52187ab38"
	board.blackMove = true
	board.multi = "Dtest"
	boards["Dtest"] = &board
	bcy = gobcy.API{"TESTTOKEN", "bcy", "test"}
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/games/", gameHandler)
	http.HandleFunc("/new/", newGameHandler)
	http.ListenAndServe(":80", nil)
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

func moveHandler(w http.ResponseWriter, r *http.Request, board *Gob) {
	//Get move and signature
	//Verify move/signature
	//If verified, set board, update memory
	//Otherwise, revert back to board setting
	f := r.FormValue
	raw := f("orig-message")
	rawmove := strings.Split(raw, "-")
	xmove, _ := strconv.Atoi(rawmove[1])
	ymove, _ := strconv.Atoi(rawmove[2])
	rawmsg := f("signed-move")
	blackVerify, _ := verifyMsg(board.blackPK, rawmsg, raw)
	whiteVerify, _ := verifyMsg(board.whitePK, rawmsg, raw)
	if board.blackMove && rawmove[0] != "black" {
		http.Error(w, "Not black's turn", http.StatusInternalServerError)
		return
	}
	if !board.blackMove && rawmove[0] != "white" {
		http.Error(w, "Not white's turn", http.StatusInternalServerError)
		return
	}
	if board.blackMove && blackVerify {
		err := board.state.SetB(xmove, ymove)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if !board.blackMove && whiteVerify {
		err := board.state.SetW(xmove, ymove)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Bad signature", http.StatusInternalServerError)
		return
	}
	if board.blackMove {
		board.blackMove = false
	} else {
		board.blackMove = true
	}
	http.Redirect(w, r, "/games/"+board.multi, http.StatusFound)
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
	pair, err := bcy.GenAddrKeychain()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.serverPub = pair.Public
	board.serverPriv = pair.Private
	//Simulate funding multisig wallet from server
	_, err = bcy.Faucet(pair, 5e6)
	txskel, err := gobcy.TempMultiTX(pair.Address, "", 4e6, 2, []string{board.serverPub, board.blackPK, board.whitePK})
	wip, err := bcy.NewTX(txskel, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Make temporary array of keys, sign transaction
	tempPriv := make([]string, len(wip.ToSign))
	for i := range tempPriv {
		tempPriv[i] = pair.Private
	}
	err = wip.Sign(tempPriv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Send transaction
	tx, err := bcy.SendTX(wip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.multi = tx.Trans.Outputs[0].Addresses[0]
	boards[board.multi] = &board
	http.Redirect(w, r, "/games/"+board.multi, http.StatusFound)
	return
}
