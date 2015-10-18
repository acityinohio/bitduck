package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/acityinohio/baduk"
	"github.com/blockcypher/gobcy"
)

type Gob struct {
	multi     string
	blackPK   string
	whitePK   string
	blackMove bool
	txskel    gobcy.TXSkel
	state     baduk.Board
}

var templates = template.Must(template.ParseGlob("templates/*"))

//Keeping it all in memory
var boards map[string]*Gob
var bcy gobcy.API

func init() {
	boards = make(map[string]*Gob)
	bcy = gobcy.API{"TESTTOKEN", "bcy", "test"}
}

func main() {
	http.HandleFunc("/", indexHandler)
	//http.HandleFunc("/games/", gameHandler)
	http.HandleFunc("/sign/", signHandler)
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

/* func gameHandler(w http.ResponseWriter, r *http.Request) {
	multi := r.URL.Path[len("/games/"):]
	board, ok := boards[multi]
	if !ok {
		http.Error(w, "Game does not exist at that address", http.StatusInternalServerError)
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
}*/

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
	wager, err := strconv.Atoi(f("wager"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.state.Init(sz)
	board.blackPK = f("blackPK")
	board.whitePK = f("whitePK")
	pubkeys := []string{board.blackPK, board.whitePK}
	board.blackMove = true
	//Generate Multisig Address for this board
	keychain, err := bcy.GenAddrMultisig(gobcy.AddrKeychain{PubKeys: pubkeys, ScriptType: "multisig-2-of-2"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.multi = keychain.Address
	//Fund Multisig with Faucet (this can be improved!)
	_, err = bcy.Faucet(gobcy.AddrKeychain{Address: board.multi}, wager)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//Setup Multisig Transaction with OP_RETURN(bitduckSIZE)
	//note that api protections mean that OP_RETURN needs to burn at least 1 satoshi
	temptx, err := gobcy.TempMultiTX("", board.multi, wager-1, 2, pubkeys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	opreturn := buildNullData("bitduck" + f("size"))
	temptx.Outputs = append(temptx.Outputs, opreturn)
	txskel, err := bcy.NewTX(temptx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.txskel = txskel
	boards[board.multi] = &board
	//Redirect to Sign Handler
	http.Redirect(w, r, "/sign/"+board.multi, http.StatusFound)
	return
}

func signHandler(w http.ResponseWriter, r *http.Request) {
	multi := r.URL.Path[len("/sign/"):]
	board, ok := boards[multi]
	if !ok {
		http.Error(w, "Game does not exist at that address", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%+v", board.txskel)
	/*if r.Method == "POST" {
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
	}*/
}

func buildNullData(data string) (opreturn gobcy.TXOutput) {
	//set value to one
	opreturn.Value = 1
	//set script type
	opreturn.ScriptType = "null-data"
	//manually craft OP_RETURN byte array with ugly one-liner
	raw := append([]byte{106, byte(len([]byte(data)))}, []byte(data)...)
	opreturn.Script = hex.EncodeToString(raw)
	return
}
