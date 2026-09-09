/**
 * CYBERSNAKE - HTML5 Canvas WebSocket Client
 * Server-authoritative, independent render rate with smooth interpolation & camera follow.
 */

(function () {
    'use strict';

    // Game Constants & Configuration
    const GRID_SIZE = 80;
    const FOOD_RADIUS = 7;
    const SUPER_FOOD_RADIUS = 12;
    const SNAKE_RADIUS = 14;

    // DOM Elements
    const canvas = document.getElementById('gameCanvas');
    const ctx = canvas.getContext('2d');
    const minimapCanvas = document.getElementById('minimapCanvas');
    const miniCtx = minimapCanvas.getContext('2d');

    const menuOverlay = document.getElementById('menuOverlay');
    const gameOverOverlay = document.getElementById('gameOverOverlay');
    const endGameCard = document.getElementById('endGameCard');
    const endGameBadge = document.getElementById('endGameBadge');
    const endGameTitle = document.getElementById('endGameTitle');
    const respawnBtnText = document.getElementById('respawnBtnText');
    const joinForm = document.getElementById('joinForm');
    const playerNameInput = document.getElementById('playerName');
    const playBtn = document.getElementById('playBtn');
    const respawnBtn = document.getElementById('respawnBtn');

    const connDot = document.getElementById('connDot');
    const connText = document.getElementById('connText');
    const hud = document.getElementById('hud');
    const scoreValue = document.getElementById('scoreValue');
    const lengthValue = document.getElementById('lengthValue');
    const playersValue = document.getElementById('playersValue');
    const leaderboardList = document.getElementById('leaderboardList');
    const finalScore = document.getElementById('finalScore');
    const deathReason = document.getElementById('deathReason');

    // Networking State
    let socket = null;
    let myPlayerId = null;
    let worldWidth = 5000;
    let worldHeight = 5000;
    let maxCanvasLength = 200;
    let isConnected = false;
    let inGame = false;

    // World & Entity State
    let currentSnakes = [];
    let currentFood = [];
    let previousState = null;
    let latestState = null;
    let stateReceiveTime = 0;
    let previousStateTime = 0;

    // Camera State
    const camera = {
        x: 2500,
        y: 2500,
        targetX: 2500,
        targetY: 2500,
        smooth: 0.15
    };

    // Resize handling
    function resizeCanvas() {
        canvas.width = window.innerWidth;
        canvas.height = window.innerHeight;
    }
    window.addEventListener('resize', resizeCanvas);
    resizeCanvas();

    // WebSocket Setup
    function connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        connText.innerText = 'Connecting to server...';
        connDot.className = 'status-dot';

        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            isConnected = true;
            connText.innerText = 'Connected! Ready to play.';
            connDot.className = 'status-dot online';
        };

        socket.onmessage = (event) => {
            // Handle multiple newline-delimited payloads in a single packet if merged
            const lines = event.data.split('\n');
            for (const line of lines) {
                if (!line.trim()) continue;
                try {
                    const msg = JSON.parse(line);
                    handleMessage(msg);
                } catch (e) {
                    console.error('Failed to parse WebSocket message', e);
                }
            }
        };

        socket.onclose = () => {
            isConnected = false;
            connText.innerText = 'Disconnected. Reconnecting in 2s...';
            connDot.className = 'status-dot';
            setTimeout(connectWebSocket, 2000);
        };

        socket.onerror = (err) => {
            console.error('WebSocket Error:', err);
        };
    }

    function handleMessage(msg) {
        switch (msg.type) {
            case 'welcome':
                myPlayerId = msg.playerId;
                worldWidth = msg.worldWidth;
                worldHeight = msg.worldHeight;
                if (msg.maxCanvasLength) {
                    maxCanvasLength = msg.maxCanvasLength;
                }
                break;

            case 'state':
                previousState = latestState;
                previousStateTime = stateReceiveTime;
                latestState = msg;
                stateReceiveTime = performance.now();

                currentSnakes = msg.snakes || [];
                currentFood = msg.food || [];

                updateHUD();
                break;

            case 'game_over':
                showEndScreen(msg.finalScore, msg.reason, false);
                break;

            case 'victory':
                showEndScreen(msg.finalScore, msg.reason, true);
                break;
        }
    }

    // Input Handling
    const KEY_DIRS = {
        'ArrowUp': 'up',
        'KeyW': 'up',
        'ArrowDown': 'down',
        'KeyS': 'down',
        'ArrowLeft': 'left',
        'KeyA': 'left',
        'ArrowRight': 'right',
        'KeyD': 'right'
    };

    let lastSentDir = '';

    window.addEventListener('keydown', (e) => {
        if (!inGame || !socket || socket.readyState !== WebSocket.OPEN) return;

        const dir = KEY_DIRS[e.code];
        if (!dir) return;

        // Prevent browser scrolling with arrow keys
        if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Space'].includes(e.code)) {
            e.preventDefault();
        }

        // Send immediately to server
        if (dir !== lastSentDir) {
            lastSentDir = dir;
            socket.send(JSON.stringify({
                type: 'input',
                direction: dir
            }));
        }
    });

    // Join / Play logic
    function joinGame() {
        if (!isConnected || !socket) return;

        const name = playerNameInput.value.trim() || 'Snake #' + Math.floor(Math.random() * 900 + 100);
        socket.send(JSON.stringify({
            type: 'join',
            name: name
        }));

        menuOverlay.classList.add('hidden');
        gameOverOverlay.classList.add('hidden');
        hud.classList.remove('hidden');
        minimapCanvas.classList.remove('hidden');
        inGame = true;
    }

    playBtn.addEventListener('click', joinGame);
    respawnBtn.addEventListener('click', joinGame);
    playerNameInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') joinGame();
    });

    function showEndScreen(score, reason, isVictory) {
        inGame = false;
        finalScore.innerText = score;
        deathReason.innerText = reason || (isVictory ? 'Victory achieved!' : 'Game over');

        if (isVictory) {
            endGameCard.classList.add('victory-card');
            endGameBadge.innerText = '👑';
            endGameBadge.classList.add('victory-badge');
            endGameTitle.innerText = 'VICTORY!';
            endGameTitle.classList.add('victory-text');
            respawnBtnText.innerText = 'PLAY AGAIN';
        } else {
            endGameCard.classList.remove('victory-card');
            endGameBadge.innerText = '💥';
            endGameBadge.classList.remove('victory-badge');
            endGameTitle.innerText = 'SNAKE TERMINATED';
            endGameTitle.classList.remove('victory-text');
            respawnBtnText.innerText = 'RESPAWN';
        }

        gameOverOverlay.classList.remove('hidden');
    }

    // HUD & Leaderboard update
    function updateHUD() {
        playersValue.innerText = currentSnakes.length;

        const mySnake = currentSnakes.find(s => s.id === myPlayerId);
        if (mySnake) {
            scoreValue.innerText = mySnake.score;
            const curLen = mySnake.body ? mySnake.body.length : 1;
            lengthValue.innerText = `${curLen}/${maxCanvasLength}`;
        }

        // Sort leaderboard by score descending
        const sorted = [...currentSnakes].sort((a, b) => b.score - a.score).slice(0, 7);
        leaderboardList.innerHTML = '';
        sorted.forEach(s => {
            const li = document.createElement('li');
            const nameSpan = document.createElement('span');
            nameSpan.className = 'leaderboard-name' + (s.id === myPlayerId ? ' is-me' : '');
            nameSpan.innerText = s.name;

            const scoreSpan = document.createElement('span');
            scoreSpan.className = 'leaderboard-score';
            scoreSpan.innerText = s.score;

            li.appendChild(nameSpan);
            li.appendChild(scoreSpan);
            leaderboardList.appendChild(li);
        });
    }

    // Render Loop (Independent 60-144+ FPS via requestAnimationFrame)
    function render() {
        requestAnimationFrame(render);

        const now = performance.now();
        const screenW = canvas.width;
        const screenH = canvas.height;

        // Locate local player snake
        const mySnake = currentSnakes.find(s => s.id === myPlayerId && s.alive);
        if (mySnake) {
            camera.targetX = mySnake.x;
            camera.targetY = mySnake.y;
        }

        // Camera smooth dampening
        camera.x += (camera.targetX - camera.x) * camera.smooth;
        camera.y += (camera.targetY - camera.y) * camera.smooth;

        // Clear background
        ctx.fillStyle = '#08090d';
        ctx.fillRect(0, 0, screenW, screenH);

        ctx.save();
        // Transform canvas by camera offset
        const offsetX = screenW / 2 - camera.x;
        const offsetY = screenH / 2 - camera.y;
        ctx.translate(offsetX, offsetY);

        // 1. Draw Grid
        drawGrid(screenW, screenH);

        // 2. Draw World Boundaries
        drawWorldBounds();

        // 3. Draw Food
        drawFood(now);

        // 4. Draw Snakes
        drawSnakes();

        ctx.restore();

        // 5. Draw Minimap Radar
        drawMinimap();
    }

    function drawGrid(screenW, screenH) {
        ctx.lineWidth = 1;
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.04)';

        const startX = Math.max(0, Math.floor((camera.x - screenW / 2) / GRID_SIZE) * GRID_SIZE);
        const endX = Math.min(worldWidth, Math.ceil((camera.x + screenW / 2) / GRID_SIZE) * GRID_SIZE);
        const startY = Math.max(0, Math.floor((camera.y - screenH / 2) / GRID_SIZE) * GRID_SIZE);
        const endY = Math.min(worldHeight, Math.ceil((camera.y + screenH / 2) / GRID_SIZE) * GRID_SIZE);

        ctx.beginPath();
        for (let x = startX; x <= endX; x += GRID_SIZE) {
            ctx.moveTo(x, startY);
            ctx.lineTo(x, endY);
        }
        for (let y = startY; y <= endY; y += GRID_SIZE) {
            ctx.moveTo(startX, y);
            ctx.lineTo(endX, y);
        }
        ctx.stroke();
    }

    function drawWorldBounds() {
        ctx.strokeStyle = '#ff0055';
        ctx.lineWidth = 6;
        ctx.shadowColor = '#ff0055';
        ctx.shadowBlur = 15;
        ctx.strokeRect(0, 0, worldWidth, worldHeight);

        // Reset shadow
        ctx.shadowBlur = 0;
    }

    function drawFood(now) {
        for (const f of currentFood) {
            const isSuper = f.type === 1;
            const r = isSuper ? SUPER_FOOD_RADIUS : FOOD_RADIUS;

            // Subtle pulsating glow
            const pulse = Math.sin(now * 0.005 + f.id) * 2;
            const drawRadius = Math.max(3, r + pulse);

            ctx.beginPath();
            ctx.arc(f.x, f.y, drawRadius, 0, Math.PI * 2);

            if (isSuper) {
                ctx.fillStyle = '#ffe600';
                ctx.shadowColor = '#ffe600';
                ctx.shadowBlur = 12;
            } else {
                ctx.fillStyle = '#00d4ff';
                ctx.shadowColor = '#00d4ff';
                ctx.shadowBlur = 8;
            }
            ctx.fill();
        }
        ctx.shadowBlur = 0;
    }

    function drawSnakes() {
        for (const snake of currentSnakes) {
            if (!snake.body || snake.body.length === 0 || !snake.alive) continue;

            const isMe = snake.id === myPlayerId;
            const color = snake.color || '#00ff88';

            // Draw Body Segments (from tail to neck)
            for (let i = snake.body.length - 1; i >= 1; i--) {
                const seg = snake.body[i];
                const ratio = (snake.body.length - i) / snake.body.length;
                const segRadius = Math.max(8, SNAKE_RADIUS * (0.65 + 0.35 * ratio));

                ctx.beginPath();
                ctx.arc(seg.x, seg.y, segRadius, 0, Math.PI * 2);
                ctx.fillStyle = color;
                ctx.globalAlpha = 0.85;
                ctx.fill();
            }

            ctx.globalAlpha = 1.0;

            // Draw Head
            const head = snake.body[0];
            ctx.beginPath();
            ctx.arc(head.x, head.y, SNAKE_RADIUS + 2, 0, Math.PI * 2);
            ctx.fillStyle = color;
            ctx.shadowColor = color;
            ctx.shadowBlur = isMe ? 18 : 10;
            ctx.fill();
            ctx.shadowBlur = 0;

            // Draw Head Eyes
            drawEyes(head, snake.direction);

            // Draw Player Name Tag above head
            ctx.font = 'bold 13px Outfit, sans-serif';
            ctx.textAlign = 'center';
            ctx.fillStyle = isMe ? '#00ff88' : '#f0f2f5';
            ctx.shadowColor = 'rgba(0,0,0,0.8)';
            ctx.shadowBlur = 6;
            ctx.fillText(snake.name + (isMe ? ' (You)' : ''), head.x, head.y - 22);
            ctx.shadowBlur = 0;
        }
    }

    function drawEyes(head, dir) {
        let eyeOffset1 = { x: 0, y: 0 };
        let eyeOffset2 = { x: 0, y: 0 };
        const fwd = 5;
        const side = 6;

        switch (dir) {
            case 'up':
                eyeOffset1 = { x: -side, y: -fwd };
                eyeOffset2 = { x: side, y: -fwd };
                break;
            case 'down':
                eyeOffset1 = { x: -side, y: fwd };
                eyeOffset2 = { x: side, y: fwd };
                break;
            case 'left':
                eyeOffset1 = { x: -fwd, y: -side };
                eyeOffset2 = { x: -fwd, y: side };
                break;
            case 'right':
            default:
                eyeOffset1 = { x: fwd, y: -side };
                eyeOffset2 = { x: fwd, y: side };
                break;
        }

        ctx.fillStyle = '#ffffff';
        ctx.beginPath();
        ctx.arc(head.x + eyeOffset1.x, head.y + eyeOffset1.y, 3, 0, Math.PI * 2);
        ctx.arc(head.x + eyeOffset2.x, head.y + eyeOffset2.y, 3, 0, Math.PI * 2);
        ctx.fill();

        ctx.fillStyle = '#08090d';
        ctx.beginPath();
        ctx.arc(head.x + eyeOffset1.x, head.y + eyeOffset1.y, 1.5, 0, Math.PI * 2);
        ctx.arc(head.x + eyeOffset2.x, head.y + eyeOffset2.y, 1.5, 0, Math.PI * 2);
        ctx.fill();
    }

    function drawMinimap() {
        const w = minimapCanvas.width;
        const h = minimapCanvas.height;

        miniCtx.clearRect(0, 0, w, h);

        // Minimap border
        miniCtx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
        miniCtx.lineWidth = 1;
        miniCtx.strokeRect(0, 0, w, h);

        const scaleX = w / worldWidth;
        const scaleY = h / worldHeight;

        // Draw snakes as dots on radar
        for (const snake of currentSnakes) {
            if (!snake.alive) continue;
            const mx = snake.x * scaleX;
            const my = snake.y * scaleY;
            const isMe = snake.id === myPlayerId;

            miniCtx.beginPath();
            miniCtx.arc(mx, my, isMe ? 3.5 : 2, 0, Math.PI * 2);
            miniCtx.fillStyle = isMe ? '#00ff88' : (snake.color || '#ff0055');
            miniCtx.fill();
        }

        // Draw camera viewport box
        const camW = (canvas.width * scaleX);
        const camH = (canvas.height * scaleY);
        const camX = (camera.x - canvas.width / 2) * scaleX;
        const camY = (camera.y - canvas.height / 2) * scaleY;

        miniCtx.strokeStyle = 'rgba(0, 255, 136, 0.4)';
        miniCtx.strokeRect(camX, camY, camW, camH);
    }

    // Initialize application
    connectWebSocket();
    requestAnimationFrame(render);
})();
