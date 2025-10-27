let gameMode = 'twoPlayer';
let gameState = null;

// Initialize the board
function initializeBoard() {
    const board = document.getElementById('game-board');
    board.innerHTML = '';
    
    for (let row = 0; row < 6; row++) {
        for (let col = 0; col < 7; col++) {
            const cell = document.createElement('div');
            cell.className = 'cell empty';
            cell.dataset.row = row;
            cell.dataset.col = col;
            cell.addEventListener('click', () => handleCellClick(col));
            board.appendChild(cell);
        }
    }
}

// Handle cell click
async function handleCellClick(col, isComputerMove = false) {
    if (gameState.gameOver) return;
    if (!isComputerMove && gameMode === 'ai' && gameState.currentPlayer === 2) return;
    
    const response = await fetch('/api/move', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ col: col, row: 0 })
    });
    
    const result = await response.json();
    
    if (result.success) {
        updateGameState(result.gameState);
        displayStatusMessage(result.message);
        
        if (result.winner > 0) {
            showWinner(result.winner, result.message);
        } else if (result.gameState && !result.gameState.gameOver && gameMode === 'ai' && result.gameState.currentPlayer === 2) {
            // Computer's turn
            setTimeout(() => computerMove(), 500);
        }
    } else {
        displayStatusMessage(result.message);
    }
}

// Computer AI move
function computerMove() {
    const col = getBestMove();
    handleCellClick(col, true);
}

// Simple AI: try to block opponent and find winning moves
function getBestMove() {
    // First, check if computer can win
    for (let col = 0; col < 7; col++) {
        if (isValidMove(col)) {
            if (wouldWin(col, 2)) {
                return col;
            }
        }
    }
    
    // Then, check if need to block player
    for (let col = 0; col < 7; col++) {
        if (isValidMove(col)) {
            if (wouldWin(col, 1)) {
                return col;
            }
        }
    }
    
    // Otherwise, play center or random valid move
    const center = 3;
    if (isValidMove(center)) {
        return center;
    }
    
    // Find random valid move
    const validMoves = [];
    for (let col = 0; col < 7; col++) {
        if (isValidMove(col)) {
            validMoves.push(col);
        }
    }
    
    if (validMoves.length > 0) {
        return validMoves[Math.floor(Math.random() * validMoves.length)];
    }
    
    return 0;
}

// Check if a move would be valid
function isValidMove(col) {
    return col >= 0 && col < 7 && gameState.board[0][col] === 0;
}

// Check if placing a piece in a column would result in a win
function wouldWin(col, player) {
    // Create a temporary board
    const tempBoard = JSON.parse(JSON.stringify(gameState.board));
    
    // Find the row to place the piece
    let row = -1;
    for (let r = 5; r >= 0; r--) {
        if (tempBoard[r][col] === 0) {
            row = r;
            tempBoard[r][col] = player;
            break;
        }
    }
    
    if (row === -1) return false;
    
    // Check for win
    player = tempBoard[row][col];
    
    // Check horizontal
    let count = 1;
    for (let j = col - 1; j >= 0 && tempBoard[row][j] === player; j--) {
        count++;
    }
    for (let j = col + 1; j < 7 && tempBoard[row][j] === player; j++) {
        count++;
    }
    if (count >= 4) return true;
    
    // Check vertical
    count = 1;
    for (let i = row + 1; i < 6 && tempBoard[i][col] === player; i++) {
        count++;
    }
    if (count >= 4) return true;
    
    // Check diagonal (top-left to bottom-right)
    count = 1;
    for (let i = row - 1, j = col - 1; i >= 0 && j >= 0 && tempBoard[i][j] === player; i--, j--) {
        count++;
    }
    for (let i = row + 1, j = col + 1; i < 6 && j < 7 && tempBoard[i][j] === player; i++, j++) {
        count++;
    }
    if (count >= 4) return true;
    
    // Check diagonal (bottom-left to top-right)
    count = 1;
    for (let i = row + 1, j = col - 1; i < 6 && j >= 0 && tempBoard[i][j] === player; i++, j--) {
        count++;
    }
    for (let i = row - 1, j = col + 1; i >= 0 && j < 7 && tempBoard[i][j] === player; i--, j++) {
        count++;
    }
    if (count >= 4) return true;
    
    return false;
}

// Update game state
function updateGameState(newState) {
    gameState = newState;
    renderBoard();
    updatePlayerIndicator();
}

// Render board
function renderBoard() {
    const cells = document.querySelectorAll('.cell');
    cells.forEach((cell, index) => {
        const row = Math.floor(index / 7);
        const col = index % 7;
        
        cell.classList.remove('filled', 'empty');
        cell.innerHTML = '';
        
        const player = gameState.board[row][col];
        
        if (player === 0) {
            cell.classList.add('empty');
        } else {
            cell.classList.add('filled');
            const token = document.createElement('div');
            token.className = player === 1 ? 'token token-red' : 'token token-yellow';
            cell.appendChild(token);
        }
    });
}

// Update player indicator
function updatePlayerIndicator() {
    if (gameState.gameOver) {
        document.getElementById('player-turn').style.display = 'none';
        return;
    }
    
    document.getElementById('player-turn').style.display = 'block';
    document.getElementById('player-number').textContent = gameState.currentPlayer;
    
    const indicator = document.getElementById('player-indicator');
    indicator.className = 'token ' + (gameState.currentPlayer === 1 ? 'token-red' : 'token-yellow');
}

// Display status message
function displayStatusMessage(message) {
    const statusEl = document.getElementById('status-message');
    if (message && message.trim()) {
        statusEl.textContent = message;
    }
}

// Show winner overlay
function showWinner(winner, message) {
    const overlay = document.getElementById('winner-overlay');
    const messageEl = document.getElementById('winner-message');
    
    if (winner === 1) {
        messageEl.textContent = '🎉 Le Joueur Rouge gagne ! 🎉';
    } else if (winner === 2) {
        messageEl.textContent = '🎉 Le Joueur Jaune gagne ! 🎉';
    } else if (winner === 3) {
        messageEl.textContent = '🤝 Match nul ! 🤝';
    } else {
        messageEl.textContent = message;
    }
    
    overlay.classList.add('show');
}

// Hide winner overlay
function hideWinner() {
    const overlay = document.getElementById('winner-overlay');
    overlay.classList.remove('show');
}

// Mode selection handlers
document.getElementById('btnTwoPlayer').addEventListener('click', () => {
    switchMode('twoPlayer');
});

document.getElementById('btnVsComputer').addEventListener('click', () => {
    switchMode('ai');
});

async function switchMode(mode) {
    gameMode = mode;
    
    // Update button states
    const buttons = document.querySelectorAll('.mode-btn');
    buttons.forEach(btn => btn.classList.remove('active'));
    if (mode === 'twoPlayer') {
        document.getElementById('btnTwoPlayer').classList.add('active');
    } else {
        document.getElementById('btnVsComputer').classList.add('active');
    }
    
    // Start new game
    await newGame();
}

// New game handler
async function newGame() {
    hideWinner();
    
    const response = await fetch('/api/new-game', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode: gameMode })
    });
    
    const result = await response.json();
    updateGameState(result.gameState);
    displayStatusMessage('');
    
    // If AI mode and it's the computer's turn, play automatically
    if (gameMode === 'ai' && result.gameState.currentPlayer === 2) {
        setTimeout(() => computerMove(), 500);
    }
}

document.getElementById('btn-new-game').addEventListener('click', newGame);
document.getElementById('btn-play-again').addEventListener('click', newGame);

// Initialize
async function init() {
    initializeBoard();
    
    const response = await fetch('/api/game');
    gameState = await response.json();
    updateGameState(gameState);
    
    // Set initial mode
    if (gameState.mode === 'twoPlayer') {
        document.getElementById('btnTwoPlayer').classList.add('active');
    } else {
        document.getElementById('btnVsComputer').classList.add('active');
    }
    gameMode = gameState.mode;
    
    // If AI mode and it's the computer's turn, play automatically
    if (gameMode === 'ai' && gameState.currentPlayer === 2 && !gameState.gameOver) {
        setTimeout(() => computerMove(), 500);
    }
}

init();
