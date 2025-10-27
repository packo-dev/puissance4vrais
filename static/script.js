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
async function handleCellClick(col) {
    if (gameState.gameOver) return;
    if (gameMode === 'ai' && gameState.currentPlayer === 2) return;
    
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
            // Computer's turn - call AI endpoint
            setTimeout(() => computerMove(), 500);
        }
    } else {
        displayStatusMessage(result.message);
    }
}

// Computer AI move - calls server-side AI
async function computerMove() {
    const response = await fetch('/api/ai-move', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
    });
    
    const result = await response.json();
    
    if (result.success) {
        updateGameState(result.gameState);
        displayStatusMessage(result.message);
        
        if (result.winner > 0) {
            showWinner(result.winner, result.message);
        }
    }
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
}

init();
