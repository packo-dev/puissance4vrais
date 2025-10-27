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

// ============================================================================
// CONSTANTS AND CONFIGURATION
// ============================================================================

const (
	BOARD_ROWS    = 6
	BOARD_COLS    = 7
	WINNING_COUNT = 4
	PLAYER_1      = 1
	PLAYER_2      = 2
	PLAYER_DRAW   = 3
	CELL_EMPTY    = 0
)

const (
	GAME_MODE_TWO_PLAYER = "twoPlayer"
	GAME_MODE_AI         = "ai"
)

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// GameState représente l'état actuel du jeu
type GameState struct {
	Board         [BOARD_ROWS][BOARD_COLS]int // Grille de jeu 6x7
	CurrentPlayer int                          // Joueur actuel (1 ou 2)
	Mode          string                       // Mode de jeu (twoPlayer ou ai)
	GameOver      bool                         // True si la partie est terminée
	Winner        int                          // 0=none, 1=J1, 2=J2, 3=draw
	StatusMessage string                       // Message d'état affiché à l'utilisateur
}

// GameResponse structure pour les réponses API JSON
type GameResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	GameState *GameState `json:"gameState,omitempty"`
	Winner    int        `json:"winner,omitempty"`
}

// ============================================================================
// GLOBAL VARIABLES
// ============================================================================

var currentGame *GameState
var tmpl *template.Template

// ============================================================================
// INITIALIZATION
// ============================================================================

// Initialise le générateur aléatoire pour les mouvements de l'IA
func init() {
	rand.Seed(time.Now().UnixNano())
}

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Initialisation du jeu avec l'état par défaut
	initializeGame()

	// Chargement du template HTML
	loadTemplates()

	// Configuration du serveur HTTP
	setupServer()

	// Démarrage du serveur
	log.Println("🎮 Serveur démarré sur http://localhost:8080")
	log.Println("📱 Ouvrez votre navigateur et commencez à jouer !")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ============================================================================
// SETUP FUNCTIONS
// ============================================================================

func initializeGame() {
	currentGame = &GameState{
		Board:         [BOARD_ROWS][BOARD_COLS]int{},
		CurrentPlayer: PLAYER_1,
		Mode:          GAME_MODE_TWO_PLAYER,
		GameOver:      false,
		Winner:        0,
		StatusMessage: "",
	}
}

func loadTemplates() {
	var err error
	tmpl, err = template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal("❌ Erreur lors du chargement du template:", err)
	}
}

func setupServer() {
	mux := http.NewServeMux()

	// Fichiers statiques (CSS, images, etc.)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes principales du jeu
	mux.HandleFunc("/", serveIndex)
	mux.HandleFunc("/game/mode", handleModeChange)
	mux.HandleFunc("/game/move", handleMove)
	mux.HandleFunc("/game/new", handleNewGame)

	// API JSON (compatibilité ascendante)
	mux.HandleFunc("/api/game", getGameStateAPI)
	mux.HandleFunc("/api/new-game", newGameAPI)
	mux.HandleFunc("/api/move", handleMoveAPI)
	mux.HandleFunc("/api/ai-move", aiMoveAPI)

	http.DefaultServeMux = mux
}

// ============================================================================
// HTTP HANDLERS - PAGES HTML
// ============================================================================

// Affiche la page principale du jeu
func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if err := tmpl.Execute(w, currentGame); err != nil {
		log.Printf("❌ Erreur d'affichage: %v", err)
		http.Error(w, "Erreur interne", http.StatusInternalServerError)
	}
}

// Gère le changement de mode de jeu (2 joueurs / IA)
func handleModeChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	mode := r.FormValue("mode")
	startNewGame(mode)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Gère le placement d'un jeton
func handleMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Récupération et validation de la colonne
	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil || col < 0 || col >= BOARD_COLS {
		currentGame.StatusMessage = "❌ Colonne invalide"
		tmpl.Execute(w, currentGame)
		return
	}

	// Placement du jeton
	row := placePiece(col, currentGame.CurrentPlayer)
	if row == -1 {
		currentGame.StatusMessage = "❌ Colonne pleine !"
		tmpl.Execute(w, currentGame)
		return
	}

	// Vérification de la victoire ou du match nul
	checkGameEnd(row, col)

	// Gestion du tour de l'IA si nécessaire
	if !currentGame.GameOver && currentGame.Mode == GAME_MODE_AI && currentGame.CurrentPlayer == PLAYER_2 {
		time.Sleep(600 * time.Millisecond) // Petite pause pour l'effet visuel
		aiMakeMove()
	}

	tmpl.Execute(w, currentGame)
}

// Commence une nouvelle partie
func handleNewGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	mode := r.FormValue("mode")
	if mode == "" {
		mode = currentGame.Mode
	}

	startNewGame(mode)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ============================================================================
// GAME LOGIC - CORE FUNCTIONS
// ============================================================================

// Place un jeton dans la colonne spécifiée
// Retourne la ligne où le jeton a été placé, ou -1 si la colonne est pleine
func placePiece(col, player int) int {
	for row := BOARD_ROWS - 1; row >= 0; row-- {
		if currentGame.Board[row][col] == CELL_EMPTY {
			currentGame.Board[row][col] = player
			return row
		}
	}
	return -1
}

// Vérifie s'il y a un gagnant après un mouvement
func checkForWin(row, col int) int {
	player := currentGame.Board[row][col]

	// Vérification horizontale
	if count := checkDirection(row, col, 0, 1, player); count >= WINNING_COUNT {
		return player
	}

	// Vérification verticale
	if count := checkDirection(row, col, 1, 0, player); count >= WINNING_COUNT {
		return player
	}

	// Vérification diagonale (haut-gauche vers bas-droite)
	if count := checkDirection(row, col, 1, 1, player); count >= WINNING_COUNT {
		return player
	}

	// Vérification diagonale (bas-gauche vers haut-droite)
	if count := checkDirection(row, col, -1, 1, player); count >= WINNING_COUNT {
		return player
	}

	return 0
}

// Compte les jetons dans une direction
func checkDirection(row, col, dRow, dCol, player int) int {
	count := 1

	// Comptage dans un sens
	for i, j := row+dRow, col+dCol; i >= 0 && i < BOARD_ROWS && j >= 0 && j < BOARD_COLS && currentGame.Board[i][j] == player; i, j = i+dRow, j+dCol {
		count++
	}

	// Comptage dans l'autre sens
	for i, j := row-dRow, col-dCol; i >= 0 && i < BOARD_ROWS && j >= 0 && j < BOARD_COLS && currentGame.Board[i][j] == player; i, j = i-dRow, j-dCol {
		count++
	}

	return count
}

// Vérifie si le plateau est plein (match nul possible)
func isBoardFull() bool {
	for col := 0; col < BOARD_COLS; col++ {
		if currentGame.Board[0][col] == CELL_EMPTY {
			return false
		}
	}
	return true
}

// Vérifie la fin de partie (victoire ou match nul)
func checkGameEnd(row, col int) {
	winner := checkForWin(row, col)

	if winner > 0 {
		currentGame.GameOver = true
		currentGame.Winner = winner
		currentGame.StatusMessage = getWinnerMessage(winner)
	} else if isBoardFull() {
		currentGame.GameOver = true
		currentGame.Winner = PLAYER_DRAW
		currentGame.StatusMessage = "🤝 Match nul !"
	} else {
		// Changement de joueur
		if currentGame.Mode == GAME_MODE_TWO_PLAYER || (currentGame.Mode == GAME_MODE_AI && currentGame.CurrentPlayer == PLAYER_1) {
			currentGame.CurrentPlayer = PLAYER_2 + PLAYER_1 - currentGame.CurrentPlayer
		}
		currentGame.StatusMessage = ""
	}
}

// Retourne le message de victoire approprié
func getWinnerMessage(winner int) string {
	switch winner {
	case PLAYER_1:
		return "🎉 Le Joueur Rouge (Joueur 1) gagne ! 🎉"
	case PLAYER_2:
		return "🎉 Le Joueur Jaune (Joueur 2) gagne ! 🎉"
	case PLAYER_DRAW:
		return "🤝 Match nul ! Égalité parfaite ! 🤝"
	default:
		return ""
	}
}

// Initialise une nouvelle partie avec le mode spécifié
func startNewGame(mode string) {
	currentGame = &GameState{
		Board:         [BOARD_ROWS][BOARD_COLS]int{},
		CurrentPlayer: PLAYER_1,
		Mode:          mode,
		GameOver:      false,
		Winner:        0,
		StatusMessage: "",
	}
}

// ============================================================================
// AI FUNCTIONS
// ============================================================================

// Fait jouer l'IA automatiquement
func aiMakeMove() {
	col := getBestMove()
	row := placePiece(col, PLAYER_2)

	if row == -1 {
		return
	}

	checkGameEnd(row, col)

	if !currentGame.GameOver {
		currentGame.CurrentPlayer = PLAYER_1
		currentGame.StatusMessage = ""
	}
}

// Calcule le meilleur mouvement pour l'IA
func getBestMove() int {
	// Priorité 1: L'IA peut-elle gagner ?
	if col := findWinningMove(PLAYER_2); col != -1 {
		return col
	}

	// Priorité 2: Bloquer l'adversaire s'il peut gagner
	if col := findWinningMove(PLAYER_1); col != -1 {
		return col
	}

	// Priorité 3: Jouer au centre (stratégique)
	centerCol := 3
	if isValidMove(centerCol) {
		return centerCol
	}

	// Sinon: mouvement aléatoire valide
	return findRandomValidMove()
}

// Trouve un mouvement gagnant pour le joueur spécifié
func findWinningMove(player int) int {
	for _, col := range getValidMoves() {
		if wouldWin(col, player) {
			return col
		}
	}
	return -1
}

// Retourne toutes les colonnes jouables
func getValidMoves() []int {
	var moves []int
	for col := 0; col < BOARD_COLS; col++ {
		if isValidMove(col) {
			moves = append(moves, col)
		}
	}
	return moves
}

// Choisit un mouvement aléatoire parmi les mouvements valides
func findRandomValidMove() int {
	moves := getValidMoves()
	if len(moves) == 0 {
		return 0
	}
	return moves[rand.Intn(len(moves))]
}

// Vérifie si un mouvement est valide (la colonne n'est pas pleine)
func isValidMove(col int) bool {
	return col >= 0 && col < BOARD_COLS && currentGame.Board[0][col] == CELL_EMPTY
}

// Simule un mouvement et vérifie s'il serait gagnant
func wouldWin(col, player int) bool {
	// Trouve la ligne où le jeton sera placé
	row := -1
	for r := BOARD_ROWS - 1; r >= 0; r-- {
		if currentGame.Board[r][col] == CELL_EMPTY {
			row = r
			break
		}
	}

	if row == -1 {
		return false
	}

	// Simulation temporaire du mouvement
	currentGame.Board[row][col] = player
	winner := checkForWin(row, col)
	currentGame.Board[row][col] = CELL_EMPTY

	return winner == player
}

// ============================================================================
// API HANDLERS - JSON ENDPOINTS
// ============================================================================

// Retourne l'état actuel du jeu en JSON
func getGameStateAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentGame)
}

// Crée une nouvelle partie via l'API
func newGameAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	startNewGame(req.Mode)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GameResponse{
		Success:   true,
		GameState: currentGame,
	})
}

// Gère un mouvement via l'API
func handleMoveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Col int `json:"col"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	row := placePiece(req.Col, currentGame.CurrentPlayer)
	if row == -1 {
		json.NewEncoder(w).Encode(GameResponse{
			Success: false,
			Message: "Colonne pleine",
		})
		return
	}

	checkGameEnd(row, req.Col)

	var response GameResponse
	if currentGame.GameOver {
		response = GameResponse{
			Success:   true,
			Message:   currentGame.StatusMessage,
			GameState: currentGame,
			Winner:    currentGame.Winner,
		}
	} else {
		response = GameResponse{
			Success:   true,
			GameState: currentGame,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Fait jouer l'IA via l'API
func aiMoveAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	col := getBestMove()
	row := placePiece(col, PLAYER_2)

	if row == -1 {
		http.Error(w, "L'IA ne peut pas jouer", http.StatusInternalServerError)
		return
	}

	checkGameEnd(row, col)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GameResponse{
		Success:   true,
		Message:   currentGame.StatusMessage,
		GameState: currentGame,
		Winner:    currentGame.Winner,
	})
}
