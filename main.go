package main

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/acityinohio/baduk"
	"github.com/blockcypher/gobcy"
)

type Gob struct {
	multi     string
	blackPK   string
	whitePK   string
	blackMove bool
	wager     int
	txskel    gobcy.TXSkel
	state     baduk.Board
}

const FEES = 9999

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
	http.HandleFunc("/games/", gameHandler)
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
	board.wager = wager
	board.blackMove = true
	//Generate Multisig Address for this board
	keychain, err := bcy.GenAddrMultisig(gobcy.AddrKeychain{PubKeys: []string{board.blackPK, board.whitePK}, ScriptType: "multisig-2-of-2"})
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
	//Put Multisig Address in Memory
	boards[board.multi] = &board
	//Setup Multisig Transaction with OP_RETURN(bitduckSIZE)
	sendTXHandler(w, r, &board, "bitduck"+f("size"))
	return
}

func sendTXHandler(w http.ResponseWriter, r *http.Request, board *Gob, raw string) {
	//Send MultiTX TX
	//note that api protections mean that OP_RETURN needs to burn at least 1 satoshi
	temptx, err := gobcy.TempMultiTX("", board.multi, board.wager-FEES-1, 2, []string{board.blackPK, board.whitePK})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	opreturn := buildNullData(raw)
	temptx.Outputs = append(temptx.Outputs, opreturn)
	temptx.Fees = FEES
	txskel, err := bcy.NewTX(temptx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.txskel = txskel
	//Redirect to Sign Handler
	http.Redirect(w, r, "/sign/"+board.multi, http.StatusFound)
	return
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

func signHandler(w http.ResponseWriter, r *http.Request) {
	multi := r.URL.Path[len("/sign/"):]
	board, ok := boards[multi]
	if !ok {
		http.Error(w, "Game does not exist at that address", http.StatusInternalServerError)
		return
	}
	if r.Method == "POST" {
		signPostHandler(w, r, board)
		return
	}
	type signTemp struct {
		Multi  string
		ToSign string
	}
	err := templates.ExecuteTemplate(w, "sign.html", signTemp{board.multi, board.txskel.ToSign[0]})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func signPostHandler(w http.ResponseWriter, r *http.Request, board *Gob) {
	f := r.FormValue
	board.txskel.Signatures = append(board.txskel.Signatures, f("blackSig"), f("whiteSig"))
	board.txskel.PubKeys = append(board.txskel.PubKeys, board.blackPK, board.whitePK)
	finTX, err := bcy.SendTX(board.txskel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.txskel = finTX
	err = updateMove(board)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/games/"+board.multi, http.StatusFound)
	return
}

func gameHandler(w http.ResponseWriter, r *http.Request) {
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
	//Get move, send transaction
	f := r.FormValue
	raw := f("orig-message")
	rawmove := strings.Split(raw, "-")
	if board.blackMove && rawmove[0] != "black" {
		http.Error(w, "Not black's turn", http.StatusInternalServerError)
		return
	}
	if !board.blackMove && rawmove[0] != "white" {
		http.Error(w, "Not white's turn", http.StatusInternalServerError)
		return
	}
	sendTXHandler(w, r, board, raw)
	return
}

//update Board based on signed TX
func updateMove(board *Gob) (err error) {
	defer func() { board.txskel = gobcy.TXSkel{} }()
	//find rawmove in OP_RETURN
	var raw string
	for _, v := range board.txskel.Trans.Outputs {
		if v.ScriptType == "pay-to-script-hash" {
			board.wager = v.Value
		}
		if v.DataString != "" {
			raw = v.DataString
		}
	}
	//decide what to do
	if strings.HasPrefix(raw, "bitduck") || raw == "gameover" {
		return
	}
	rawmove := strings.Split(raw, "-")
	xmove, _ := strconv.Atoi(rawmove[1])
	ymove, _ := strconv.Atoi(rawmove[2])
	if board.blackMove {
		err = board.state.SetB(xmove, ymove)
		if err != nil {
			return
		}
	} else if !board.blackMove {
		err = board.state.SetW(xmove, ymove)
		if err != nil {
			return
		}
	}
	if board.blackMove {
		board.blackMove = false
	} else {
		board.blackMove = true
	}
	return
}
