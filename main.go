package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
