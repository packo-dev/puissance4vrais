package main

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// Initialize random seed for AI moves
func init() {
	rand.Seed(time.Now().UnixNano())
}

type GameState struct {
	Board         [6][7]int
	CurrentPlayer int
	Mode          string
	GameOver      bool
	Winner        int
	StatusMessage string
}

// Global game state (in production, use sessions or database)
var currentGame *GameState
var tmpl *template.Template

func main() {
	// Initialize game
	currentGame = &GameState{
		Board:         [6][7]int{},
		CurrentPlayer:  1,
		Mode:          "twoPlayer",
		GameOver:      false,
		Winner:        0,
		StatusMessage: "",
	}

	// Load templates
	var err error
	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	// Setup routes
	mux := http.NewServeMux()
	
	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	// Game routes
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/game/mode", handleModeChange)
	mux.HandleFunc("/game/move", handleMove)
	mux.HandleFunc("/game/new", handleNewGame)
	
	// API endpoints for backwards compatibility
	mux.HandleFunc("/api/game", getGameStateAPI)
	mux.HandleFunc("/api/new-game", newGameAPI)
	mux.HandleFunc("/api/move", handleMoveAPI)
	mux.HandleFunc("/api/ai-move", aiMoveAPI)

	// Start server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	tmpl.Execute(w, currentGame)
}

func handleModeChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := r.FormValue("mode")
	currentGame.Mode = mode
	currentGame.Board = [6][7]int{}
	currentGame.CurrentPlayer = 1
	currentGame.GameOver = false
	currentGame.Winner = 0
	currentGame.StatusMessage = ""

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil || col < 0 || col > 6 {
		currentGame.StatusMessage = "Colonne invalide"
		tmpl.Execute(w, currentGame)
		return
	}

	// Place piece
	row := placePiece(col, currentGame.CurrentPlayer)
	
	if row == -1 {
		currentGame.StatusMessage = "Colonne pleine"
		tmpl.Execute(w, currentGame)
		return
	}

	// Check for win
	winner := checkForWin(row, col)
	
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		currentGame.StatusMessage = getWinnerMessage(winner)
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3
		currentGame.StatusMessage = "Match nul !"
	} else {
		// Switch player (handle AI turn if needed)
		if currentGame.Mode == "twoPlayer" || (currentGame.Mode == "ai" && currentGame.CurrentPlayer == 1) {
			currentGame.CurrentPlayer = 3 - currentGame.CurrentPlayer
		}
		
		// If AI mode and player 2's turn, make AI move
		if currentGame.Mode == "ai" && currentGame.CurrentPlayer == 2 && !currentGame.GameOver {
			time.Sleep(600 * time.Millisecond) // Pause de 600ms avant que l'IA joue
			aiMakeMove()
		}
		currentGame.StatusMessage = ""
	}

	tmpl.Execute(w, currentGame)
}

func handleNewGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mode := r.FormValue("mode")
	currentGame = &GameState{
		Board:         [6][7]int{},
		CurrentPlayer:  1,
		Mode:          mode,
		GameOver:      false,
		Winner:        0,
		StatusMessage: "",
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// AI makes a move automatically
func aiMakeMove() {
	col := getBestMove()
	row := placePiece(col, 2)
	
	if row == -1 {
		return
	}

	// Check for win
	winner := checkForWin(row, col)
	
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		currentGame.StatusMessage = getWinnerMessage(winner)
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3
		currentGame.StatusMessage = "Match nul !"
	} else {
		currentGame.CurrentPlayer = 1
		currentGame.StatusMessage = ""
	}
}

// Place a piece in the lowest available row
func placePiece(col, player int) int {
	for row := 5; row >= 0; row-- {
		if currentGame.Board[row][col] == 0 {
			currentGame.Board[row][col] = player
			return row
		}
	}
	return -1
}

// Check if there's a winner
func checkForWin(row, col int) int {
	player := currentGame.Board[row][col]
	
	// Check horizontal
	count := 1
	for j := col - 1; j >= 0 && currentGame.Board[row][j] == player; j-- {
		count++
	}
	for j := col + 1; j < 7 && currentGame.Board[row][j] == player; j++ {
		count++
	}
	if count >= 4 {
		return player
	}

	// Check vertical
	count = 1
	for i := row + 1; i < 6 && currentGame.Board[i][col] == player; i++ {
		count++
	}
	if count >= 4 {
		return player
	}

	// Check diagonal (top-left to bottom-right)
	count = 1
	for i, j := row-1, col-1; i >= 0 && j >= 0 && currentGame.Board[i][j] == player; i, j = i-1, j-1 {
		count++
	}
	for i, j := row+1, col+1; i < 6 && j < 7 && currentGame.Board[i][j] == player; i, j = i+1, j+1 {
		count++
	}
	if count >= 4 {
		return player
	}

	// Check diagonal (bottom-left to top-right)
	count = 1
	for i, j := row+1, col-1; i < 6 && j >= 0 && currentGame.Board[i][j] == player; i, j = i+1, j-1 {
		count++
	}
	for i, j := row-1, col+1; i >= 0 && j < 7 && currentGame.Board[i][j] == player; i, j = i-1, j+1 {
		count++
	}
	if count >= 4 {
		return player
	}

	return 0
}

func isBoardFull() bool {
	for col := 0; col < 7; col++ {
		if currentGame.Board[0][col] == 0 {
			return false
		}
	}
	return true
}

func getWinnerMessage(winner int) string {
	switch winner {
	case 1:
		return "Le joueur rouge (Joueur 1) gagne ! 🎉"
	case 2:
		return "Le joueur jaune (Joueur 2) gagne ! 🎉"
	case 3:
		return "Match nul ! Égalité ! 🤝"
	default:
		return ""
	}
}

// Get the best move for the AI
func getBestMove() int {
	// First, check if AI can win
	if col := findWinningMove(2); col != -1 {
		return col
	}

	// Then, check if need to block player 1
	if col := findWinningMove(1); col != -1 {
		return col
	}

	// Otherwise, prefer center column if available
	center := 3
	if isValidMove(center) {
		return center
	}

	// Find first valid move randomly
	return findRandomValidMove()
}

// Find a winning move for the specified player
func findWinningMove(player int) int {
	validMoves := getValidMoves()
	for _, col := range validMoves {
		if wouldWin(col, player) {
			return col
		}
	}
	return -1
}

// Get all valid column moves
func getValidMoves() []int {
	var moves []int
	for col := 0; col < 7; col++ {
		if isValidMove(col) {
			moves = append(moves, col)
		}
	}
	return moves
}

// Find a random valid move
func findRandomValidMove() int {
	moves := getValidMoves()
	if len(moves) == 0 {
		return 0
	}
	return moves[rand.Intn(len(moves))]
}

// Check if a move is valid (column is not full)
func isValidMove(col int) bool {
	return col >= 0 && col < 7 && currentGame.Board[0][col] == 0
}

// Check if placing a piece would result in a win
func wouldWin(col, player int) bool {
	row := -1
	for r := 5; r >= 0; r-- {
		if currentGame.Board[r][col] == 0 {
			row = r
			break
		}
	}
	
	if row == -1 {
		return false
	}

	currentGame.Board[row][col] = player
	winner := checkForWin(row, col)
	currentGame.Board[row][col] = 0
	
	return winner == player
}

// API endpoints for backwards compatibility (return JSON)
type GameResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	GameState *GameState `json:"gameState,omitempty"`
	Winner    int       `json:"winner,omitempty"`
}

func getGameStateAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentGame)
}

func newGameAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Mode string }
	json.NewDecoder(r.Body).Decode(&req)
	currentGame = &GameState{Board: [6][7]int{}, CurrentPlayer: 1, Mode: req.Mode, GameOver: false, Winner: 0}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GameResponse{Success: true, GameState: currentGame})
}

func handleMoveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Col int }
	json.NewDecoder(r.Body).Decode(&req)
	row := placePiece(req.Col, currentGame.CurrentPlayer)
	if row == -1 {
		json.NewEncoder(w).Encode(GameResponse{Success: false, Message: "Column is full"})
		return
	}
	winner := checkForWin(row, req.Col)
	var response GameResponse
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		response = GameResponse{Success: true, Message: getWinnerMessage(winner), GameState: currentGame, Winner: winner}
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3
		response = GameResponse{Success: true, Message: "Match nul !", GameState: currentGame, Winner: 3}
	} else {
		if currentGame.Mode == "twoPlayer" {
			currentGame.CurrentPlayer = 3 - currentGame.CurrentPlayer
		} else if currentGame.Mode == "ai" {
			currentGame.CurrentPlayer = 3 - currentGame.CurrentPlayer
		}
		response = GameResponse{Success: true, GameState: currentGame}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func aiMoveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	col := getBestMove()
	row := placePiece(col, 2)
	if row == -1 {
		http.Error(w, "AI cannot make a move", http.StatusInternalServerError)
		return
	}
	winner := checkForWin(row, col)
	var response GameResponse
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		response = GameResponse{Success: true, Message: getWinnerMessage(winner), GameState: currentGame, Winner: winner}
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3
		response = GameResponse{Success: true, Message: "Match nul !", GameState: currentGame, Winner: 3}
	} else {
		currentGame.CurrentPlayer = 1
		response = GameResponse{Success: true, GameState: currentGame}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}