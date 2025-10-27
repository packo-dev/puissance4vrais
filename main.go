package main

import (
	"encoding/json"
	"log"
	"net/http"
	"math/rand"
	"time"
)

// Initialize random seed for AI moves
func init() {
	rand.Seed(time.Now().UnixNano())
}

type GameState struct {
	Board        [6][7]int `json:"board"` // 0=empty, 1=player1, 2=player2
	CurrentPlayer int      `json:"currentPlayer"` // 1 or 2
	Mode         string    `json:"mode"` // "ai" or "twoPlayer"
	GameOver     bool      `json:"gameOver"`
	Winner       int       `json:"winner"` // 0=none, 1=player1, 2=player2, 3=draw
}

type MoveRequest struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

type GameResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	GameState *GameState `json:"gameState,omitempty"`
	Winner    int       `json:"winner,omitempty"`
}

// Global game state (in production, use sessions or database)
var currentGame *GameState

func main() {
	// Initialize game
	currentGame = &GameState{
		Board:        [6][7]int{},
		CurrentPlayer: 1,
		Mode:         "twoPlayer",
		GameOver:     false,
		Winner:       0,
	}

	// Setup routes
	mux := http.NewServeMux()
	
	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	// Serve index.html
	mux.HandleFunc("/", serveIndex)
	
	// API endpoints
	mux.HandleFunc("/api/game", getGameState)
	mux.HandleFunc("/api/new-game", newGame)
	mux.HandleFunc("/api/move", handleMove)
	mux.HandleFunc("/api/ai-move", aiMove)
	mux.HandleFunc("/api/check-win", checkWin)

	// Start server
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	http.ServeFile(w, r, "templates/index.html")
}

func getGameState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentGame)
}

func newGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse mode from request body
	var req struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentGame = &GameState{
		Board:        [6][7]int{},
		CurrentPlayer: 1,
		Mode:         req.Mode,
		GameOver:     false,
		Winner:       0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GameResponse{
		Success:   true,
		Message:   "New game started",
		GameState: currentGame,
	})
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Place piece in the lowest available row in the column
	row := placePiece(req.Col, currentGame.CurrentPlayer)
	
	if row == -1 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GameResponse{
			Success: false,
			Message: "Column is full",
		})
		return
	}

	// Check for win
	winner := checkForWin(row, req.Col)
	
	var response GameResponse
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		response = GameResponse{
			Success:   true,
			Message:   getWinnerMessage(winner),
			GameState: currentGame,
			Winner:    winner,
		}
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3 // Draw
		response = GameResponse{
			Success:   true,
			Message:   "Match nul !",
			GameState: currentGame,
			Winner:    3,
		}
	} else {
		// Switch player
		if currentGame.Mode == "twoPlayer" {
			// In two player mode, always alternate
			currentGame.CurrentPlayer = 3 - currentGame.CurrentPlayer
		} else if currentGame.Mode == "ai" {
			// In AI mode, alternate between human (player 1) and computer (player 2)
			currentGame.CurrentPlayer = 3 - currentGame.CurrentPlayer
		}
		response = GameResponse{
			Success:   true,
			Message:   "Move successful",
			GameState: currentGame,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func checkWin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"gameOver": currentGame.GameOver,
		"winner":   currentGame.Winner,
		"message":  getWinnerMessage(currentGame.Winner),
	})
}

// Place a piece in the lowest available row
func placePiece(col, player int) int {
	for row := 5; row >= 0; row-- {
		if currentGame.Board[row][col] == 0 {
			currentGame.Board[row][col] = player
			return row
		}
	}
	return -1 // Column is full
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

// AI endpoint - plays a move for the computer
func aiMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if currentGame.Mode != "ai" || currentGame.CurrentPlayer != 2 || currentGame.GameOver {
		http.Error(w, "Invalid AI move request", http.StatusBadRequest)
		return
	}

	// Get best move
	col := getBestMove()
	
	// Place piece
	row := placePiece(col, 2)
	
	if row == -1 {
		http.Error(w, "AI cannot make a move", http.StatusInternalServerError)
		return
	}

	// Check for win
	winner := checkForWin(row, col)
	
	var response GameResponse
	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		response = GameResponse{
			Success:   true,
			Message:   getWinnerMessage(winner),
			GameState: currentGame,
			Winner:    winner,
		}
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = 3
		response = GameResponse{
			Success:   true,
			Message:   "Match nul !",
			GameState: currentGame,
			Winner:    3,
		}
	} else {
		// Switch back to player 1
		currentGame.CurrentPlayer = 1
		response = GameResponse{
			Success:   true,
			Message:   "AI move successful",
			GameState: currentGame,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	// Find the row where the piece would land
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

	// Temporarily place the piece
	currentGame.Board[row][col] = player
	
	// Check if this creates a win
	winner := checkForWin(row, col)
	
	// Restore the cell
	currentGame.Board[row][col] = 0
	
	return winner == player
}
