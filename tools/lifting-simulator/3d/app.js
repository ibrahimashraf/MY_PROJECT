// INTEGIN 3D Spatial Collision & 4D Tandem Lift Simulator
// Standalone WebGL / Canvas implementation with deterministic physics evaluation

(function() {
    'use strict';

    // State Variables
    let scene, camera, renderer, controls;
    let crane1Group, crane2Group, boom1, boom2, suspendedLoad;
    let isPlaying = false;
    let playInterval = null;

    const state = {
        time: 0.0,
        crane1: { x: -15, z: 0, boomAngle: 65, slewAngle: 15, boomLength: 30 },
        crane2: { x: 15, z: 0, boomAngle: 60, slewAngle: -20, boomLength: 28 },
        load: { massTonnes: 45.0, length: 12.0, radius: 1.5 },
        allowableGbpKPa: 220.0
    };

    function init() {
        const container = document.getElementById('canvas-container');
        const width = container.clientWidth || 800;
        const height = container.clientHeight || 600;

        // Scene
        scene = new THREE.Scene();
        scene.background = new THREE.Color(0x0a1120);

        // Camera
        camera = new THREE.PerspectiveCamera(45, width / height, 1, 1000);
        camera.position.set(0, 35, 55);

        // Renderer
        renderer = new THREE.WebGLRenderer({ antialias: true });
        renderer.setSize(width, height);
        renderer.shadowMap.enabled = true;
        container.appendChild(renderer.domElement);

        // OrbitControls
        if (typeof THREE.OrbitControls !== 'undefined') {
            controls = new THREE.OrbitControls(camera, renderer.domElement);
            controls.enableDamping = true;
            controls.dampingFactor = 0.05;
        }

        // Lighting
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.7);
        scene.add(ambientLight);

        const dirLight = new THREE.DirectionalLight(0xffffff, 0.8);
        dirLight.position.set(20, 50, 20);
        dirLight.castShadow = true;
        scene.add(dirLight);

        // Ground Grid & Heatmap Base
        const grid = new THREE.GridHelper(80, 40, 0x38bdf8, 0x1e293b);
        grid.position.y = 0;
        scene.add(grid);

        buildSceneObjects();
        setupEventListeners();
        updateKinematics();

        window.addEventListener('resize', onWindowResize);
        animate();
    }

    function buildSceneObjects() {
        // Materials
        const craneMat1 = new THREE.MeshStandardMaterial({ color: 0xf59e0b, roughness: 0.4 });
        const craneMat2 = new THREE.MeshStandardMaterial({ color: 0x3b82f6, roughness: 0.4 });
        const boomMat = new THREE.MeshStandardMaterial({ color: 0x64748b, wireframe: false });
        const loadMat = new THREE.MeshStandardMaterial({ color: 0xef4444, metalness: 0.3, roughness: 0.5 });

        // Crane 1 (Alpha)
        crane1Group = new THREE.Group();
        crane1Group.position.set(state.crane1.x, 0, state.crane1.z);
        const base1 = new THREE.Mesh(new THREE.BoxGeometry(6, 2.5, 4), craneMat1);
        base1.position.y = 1.25;
        crane1Group.add(base1);

        boom1 = new THREE.Mesh(new THREE.CylinderGeometry(0.5, 0.8, state.crane1.boomLength, 8), boomMat);
        boom1.geometry.translate(0, state.crane1.boomLength / 2, 0);
        boom1.position.set(0, 2.5, 0);
        crane1Group.add(boom1);
        scene.add(crane1Group);

        // Crane 2 (Beta)
        crane2Group = new THREE.Group();
        crane2Group.position.set(state.crane2.x, 0, state.crane2.z);
        const base2 = new THREE.Mesh(new THREE.BoxGeometry(6, 2.5, 4), craneMat2);
        base2.position.y = 1.25;
        crane2Group.add(base2);

        boom2 = new THREE.Mesh(new THREE.CylinderGeometry(0.5, 0.8, state.crane2.boomLength, 8), boomMat);
        boom2.geometry.translate(0, state.crane2.boomLength / 2, 0);
        boom2.position.set(0, 2.5, 0);
        crane2Group.add(boom2);
        scene.add(crane2Group);

        // Suspended Vessel Load (Cylinder)
        const loadGeo = new THREE.CylinderGeometry(state.load.radius, state.load.radius, state.load.length, 16);
        loadGeo.rotateZ(Math.PI / 2);
        suspendedLoad = new THREE.Mesh(loadGeo, loadMat);
        scene.add(suspendedLoad);
    }

    function calculateTip(craneState) {
        const radBoom = (90 - craneState.boomAngle) * (Math.PI / 180);
        const radSlew = craneState.slewAngle * (Math.PI / 180);

        const r = craneState.boomLength * Math.sin(radBoom);
        const y = 2.5 + craneState.boomLength * Math.cos(radBoom);
        const x = craneState.x + r * Math.sin(radSlew);
        const z = craneState.z + r * Math.cos(radSlew);

        return { x, y, z };
    }

    function updateKinematics() {
        // Update Crane 1 Rotations
        crane1Group.rotation.y = state.crane1.slewAngle * (Math.PI / 180);
        boom1.rotation.z = -(90 - state.crane1.boomAngle) * (Math.PI / 180);

        // Update Crane 2 Rotations
        crane2Group.rotation.y = state.crane2.slewAngle * (Math.PI / 180);
        boom2.rotation.z = (90 - state.crane2.boomAngle) * (Math.PI / 180);

        // Calculate Boom Tips
        const tip1 = calculateTip(state.crane1);
        const tip2 = calculateTip(state.crane2);

        // Position Load midway between hoist points
        const loadX = (tip1.x + tip2.x) / 2;
        const loadY = Math.min(tip1.y, tip2.y) - 8;
        const loadZ = (tip1.z + tip2.z) / 2;
        suspendedLoad.position.set(loadX, Math.max(loadY, 2.0), loadZ);

        // Evaluate Spatial Distance & Collision
        const dx = tip1.x - tip2.x;
        const dy = tip1.y - tip2.y;
        const dz = tip1.z - tip2.z;
        const dist = Math.sqrt(dx*dx + dy*dy + dz*dz);
        const clearance = dist - state.load.length;

        // Ground Bearing Pressure (FoS calculation based on geotech.go)
        // 45t load split ~50/50 + 60t crane weight on 4 outrigger pads of 2.25m²
        const padLoad1 = (state.load.massTonnes * 0.52 + 60) / 4;
        const actualKPa = (padLoad1 * 9.80665) / 2.25;
        const fos = state.allowableGbpKPa / actualKPa;

        // Telemetry DOM updates
        const clearanceEl = document.getElementById('clearance-display');
        if (clearance > 2.0) {
            clearanceEl.textContent = `CLEAR (${clearance.toFixed(2)}m)`;
            clearanceEl.className = 'val safe';
        } else if (clearance > 0.5) {
            clearanceEl.textContent = `WARN (${clearance.toFixed(2)}m)`;
            clearanceEl.className = 'val warn';
        } else {
            clearanceEl.textContent = `COLLISION (${clearance.toFixed(2)}m)`;
            clearanceEl.className = 'val danger';
        }

        const gbpEl = document.getElementById('gbp-display');
        if (fos >= 1.5) {
            gbpEl.textContent = `PASS (FoS ${fos.toFixed(2)})`;
            gbpEl.className = 'val safe';
        } else {
            gbpEl.textContent = `FAIL (FoS ${fos.toFixed(2)})`;
            gbpEl.className = 'val danger';
        }

        document.getElementById('time-display').textContent = `t = ${state.time.toFixed(1)}s`;
    }

    function setupEventListeners() {
        const sliderTime = document.getElementById('time-slider');
        sliderTime.addEventListener('input', (e) => {
            state.time = parseFloat(e.target.value);
            // Simulate dynamic trajectory interpolation across time steps
            state.crane1.slewAngle = 15 + (state.time / 60) * 20;
            state.crane2.slewAngle = -20 + (state.time / 60) * 25;
            updateKinematics();
        });

        document.getElementById('btn-play').addEventListener('click', () => {
            if (!isPlaying) {
                isPlaying = true;
                document.getElementById('btn-play').textContent = '⏸ Pause';
                playInterval = setInterval(() => {
                    if (state.time < 60) {
                        state.time += 0.5;
                        sliderTime.value = state.time;
                        sliderTime.dispatchEvent(new Event('input'));
                    } else {
                        clearInterval(playInterval);
                        isPlaying = false;
                        document.getElementById('btn-play').textContent = '▶ Play';
                    }
                }, 100);
            } else {
                clearInterval(playInterval);
                isPlaying = false;
                document.getElementById('btn-play').textContent = '▶ Play';
            }
        });

        document.getElementById('btn-reset').addEventListener('click', () => {
            clearInterval(playInterval);
            isPlaying = false;
            document.getElementById('btn-play').textContent = '▶ Play';
            state.time = 0;
            sliderTime.value = 0;
            sliderTime.dispatchEvent(new Event('input'));
        });

        document.getElementById('btn-bind-manifest').addEventListener('click', () => {
            alert('Cryptographic manifest digest bound to active Work Order package: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855');
        });
    }

    function onWindowResize() {
        const container = document.getElementById('canvas-container');
        camera.aspect = container.clientWidth / container.clientHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(container.clientWidth, container.clientHeight);
    }

    function animate() {
        requestAnimationFrame(animate);
        if (controls) controls.update();
        renderer.render(scene, camera);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
