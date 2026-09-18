// INTEGIN 3D Spatial Collision & 4D Tandem Lift Simulator
// Standalone WebGL / Canvas implementation with deterministic physics evaluation

(function() {
    'use strict';

    // State Variables
    let scene, camera, renderer, controls;
    let crane1Group, crane2Group, boom1, boom2, suspendedLoad;
    let hook1Mesh, hook2Mesh, slingLines, hoistLines;
    let isPlaying = false;
    let playInterval = null;
    let selectedId = null;
    let viewMode = 'fade';
    let inspectorTab = 'Overview';
    let verdictTimer = null;
    let verdictOnline = false;
    const live = { radius1: 0, radius2: 0, share1: 52, clearance: '', fos: 0, fosText: '' };

    const state = {
        time: 0.0,
        window: 60.0,
        crane1: { x: -15, z: 0, boomAngle: 65, slewAngle: 15, boomLength: 30 },
        crane2: { x: 15, z: 0, boomAngle: 60, slewAngle: -20, boomLength: 28 },
        load: { massTonnes: 45.0, length: 12.0, radius: 1.5 },
        allowableGbpKPa: 220.0,
        // User-supplied plan data. Defaults mirror the demo scenario; the
        // engine treats every value as untrusted input and gates it.
        plan: {
            cap1: 40, cap2: 40, counterweight: 20, spread: 8.5,
            chassis1: 60, chassis2: 60,
            slingAngle: 60, slingMBL: 30, rigging: 2,
            matArea: 3.0, cogOffset: -1
        }
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

        // Renderer with graceful fallback for legacy / software-only devices
        try {
            renderer = new THREE.WebGLRenderer({ antialias: true, failIfMajorPerformanceCaveat: false });
        } catch (e) {
            console.warn('Hardware WebGL failed, falling back without antialiasing:', e);
            try {
                renderer = new THREE.WebGLRenderer({ antialias: false, powerPreference: 'low-power' });
            } catch (err) {
                console.error('WebGL unavailable on this device/browser:', err);
                const notice = document.createElement('div');
                notice.style.cssText = 'color:#f87171;padding:24px;text-align:center;font-family:sans-serif;background:#1e293b;border-radius:8px;margin:20px;';
                notice.innerHTML = '<h3>⚠️ Hardware Acceleration / 3D Graphics Unavailable</h3><p>Your browser or OS does not currently provide WebGL/WebGPU acceleration. Telemetry and calculations remain 100% active in the control panels.</p>';
                container.appendChild(notice);
                return;
            }
        }
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
        buildSystemsList();
        
        // Initialize timeline to stage 0
        const sliderTime = document.getElementById('time-slider');
        if (sliderTime) {
            sliderTime.dispatchEvent(new Event('input'));
        } else {
            updateKinematics();
        }

        window.addEventListener('resize', onWindowResize);
        animate();
    }

    function makeBoomMesh(len, mat) {
        const m = new THREE.Mesh(new THREE.CylinderGeometry(0.5, 0.8, len, 8), mat);
        m.geometry.translate(0, len / 2, 0);
        return m;
    }

    function rebuildBoom(mesh, len) {
        mesh.geometry.dispose();
        mesh.geometry = new THREE.CylinderGeometry(0.5, 0.8, len, 8);
        mesh.geometry.translate(0, len / 2, 0);
    }

    function rebuildLoad() {
        suspendedLoad.geometry.dispose();
        const g = new THREE.CylinderGeometry(state.load.radius, state.load.radius, state.load.length, 16);
        g.rotateZ(Math.PI / 2);
        suspendedLoad.geometry = g;
    }

    function buildSceneObjects() {
        // Materials (cloned per component to prevent state bleed)
        const craneMat1 = new THREE.MeshStandardMaterial({ color: 0xf59e0b, roughness: 0.4 });
        const craneMat2 = new THREE.MeshStandardMaterial({ color: 0x3b82f6, roughness: 0.4 });
        const boomMat1 = new THREE.MeshStandardMaterial({ color: 0x64748b, roughness: 0.5 });
        const boomMat2 = new THREE.MeshStandardMaterial({ color: 0x64748b, roughness: 0.5 });
        const loadMat = new THREE.MeshStandardMaterial({ color: 0xef4444, metalness: 0.3, roughness: 0.5 });

        // Crane 1 (Alpha)
        crane1Group = new THREE.Group();
        crane1Group.position.set(state.crane1.x, 0, state.crane1.z);
        crane1Group.userData.nodeId = 'CRN-ALPHA';
        const base1 = new THREE.Mesh(new THREE.BoxGeometry(6, 2.5, 4), craneMat1);
        base1.position.y = 1.25;
        base1.userData.nodeId = 'CRN-ALPHA';
        crane1Group.add(base1);

        boom1 = makeBoomMesh(state.crane1.boomLength, boomMat1);
        boom1.position.set(0, 2.5, 0);
        boom1.userData.nodeId = 'CRN-ALPHA';
        crane1Group.add(boom1);
        addMatRow(crane1Group);
        scene.add(crane1Group);

        // Crane 2 (Beta)
        crane2Group = new THREE.Group();
        crane2Group.position.set(state.crane2.x, 0, state.crane2.z);
        crane2Group.userData.nodeId = 'CRN-BETA';
        const base2 = new THREE.Mesh(new THREE.BoxGeometry(6, 2.5, 4), craneMat2);
        base2.position.y = 1.25;
        base2.userData.nodeId = 'CRN-BETA';
        crane2Group.add(base2);

        boom2 = makeBoomMesh(state.crane2.boomLength, boomMat2);
        boom2.position.set(0, 2.5, 0);
        boom2.userData.nodeId = 'CRN-BETA';
        crane2Group.add(boom2);
        addMatRow(crane2Group);
        scene.add(crane2Group);

        // Suspended Vessel Load (Cylinder)
        const loadGeo = new THREE.CylinderGeometry(state.load.radius, state.load.radius, state.load.length, 16);
        loadGeo.rotateZ(Math.PI / 2);
        suspendedLoad = new THREE.Mesh(loadGeo, loadMat);
        suspendedLoad.userData.nodeId = 'VESSEL-45T';
        scene.add(suspendedLoad);

        // Hook blocks ride below tips on hoist lines
        const hookMat = new THREE.MeshStandardMaterial({ color: 0xfacc15, roughness: 0.5 });
        hook1Mesh = new THREE.Mesh(new THREE.BoxGeometry(1.2, 1.6, 1.2), hookMat);
        hook1Mesh.userData.nodeId = 'HOOK-ALPHA';
        hook2Mesh = new THREE.Mesh(new THREE.BoxGeometry(1.2, 1.6, 1.2), hookMat.clone());
        hook2Mesh.userData.nodeId = 'HOOK-BETA';
        scene.add(hook1Mesh);
        scene.add(hook2Mesh);

        // Hoist pendant wire lines: boom tip -> hook block
        const hoistWireMat = new THREE.LineBasicMaterial({ color: 0x94a3b8 });
        hoistLines = [];
        for (let i = 0; i < 2; i++) {
            const g = new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(), new THREE.Vector3()]);
            const line = new THREE.Line(g, hoistWireMat);
            line.userData.nodeId = i === 0 ? 'HOOK-ALPHA' : 'HOOK-BETA';
            hoistLines.push(line);
            scene.add(line);
        }

        // Sling legs: hook blocks to load trunnions
        const slingMat = new THREE.LineBasicMaterial({ color: 0xe2e8f0 });
        slingLines = [];
        for (let i = 0; i < 4; i++) {
            const g = new THREE.BufferGeometry().setFromPoints([new THREE.Vector3(), new THREE.Vector3()]);
            const line = new THREE.Line(g, slingMat);
            line.userData.nodeId = i < 2 ? 'SLING-SET' : 'SHACKLE-SET';
            slingLines.push(line);
            scene.add(line);
        }

        // Pick Staging: Heavy transport low-boy trailer bed (at initial pickup location Z ≈ 12)
        const trailerMat = new THREE.MeshStandardMaterial({ color: 0x475569, roughness: 0.7 });
        const trailer = new THREE.Mesh(new THREE.BoxGeometry(14, 0.6, 3.5), trailerMat);
        trailer.position.set(0, 0.3, 12.2);
        trailer.userData.nodeId = 'SITE-EQUIP';
        scene.add(trailer);

        // Set Staging: Concrete target foundation plinths/saddles (at final set location Z ≈ 3.8)
        const foundationMat = new THREE.MeshStandardMaterial({ color: 0x94a3b8, roughness: 0.8 });
        [-4.5, 4.5].forEach((offset) => {
            const plinth = new THREE.Mesh(new THREE.BoxGeometry(2.0, 1.2, 3.2), foundationMat);
            plinth.position.set(offset, 0.6, 3.8);
            plinth.userData.nodeId = 'SITE-FOUNDATION';
            scene.add(plinth);
        });
    }

    function addMatRow(group) {
        // 3.0 m² spec pads: 1.73 m square (≈2.99 m²), schematic at spread corners.
        const matGeo = new THREE.BoxGeometry(1.73, 0.3, 1.73);
        const mat1 = new THREE.MeshStandardMaterial({ color: 0x92400e, roughness: 0.9 });
        [[-4.25, -3], [4.25, -3], [-4.25, 3], [4.25, 3]].forEach(function (p) {
            const m = new THREE.Mesh(matGeo, mat1);
            m.position.set(p[0], 0.15, p[1]);
            m.userData.nodeId = 'MAT-SYS';
            group.add(m);
        });
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
        // Update Crane 1 Rotations (pitch boom towards center)
        crane1Group.rotation.y = state.crane1.slewAngle * (Math.PI / 180);
        boom1.rotation.z = -(90 - state.crane1.boomAngle) * (Math.PI / 180);

        // Update Crane 2 Rotations (pitch boom inwards towards crane 1)
        crane2Group.rotation.y = (180 + state.crane2.slewAngle) * (Math.PI / 180);
        boom2.rotation.z = -(90 - state.crane2.boomAngle) * (Math.PI / 180);

        // Update world matrices so tip vectors match the exact mesh orientation
        crane1Group.updateMatrixWorld(true);
        crane2Group.updateMatrixWorld(true);

        // Calculate Boom Tips directly from the local boom mesh top (0, len, 0)
        const tip1Vec = new THREE.Vector3(0, state.crane1.boomLength, 0);
        boom1.localToWorld(tip1Vec);
        const tip2Vec = new THREE.Vector3(0, state.crane2.boomLength, 0);
        boom2.localToWorld(tip2Vec);

        const tip1 = { x: tip1Vec.x, y: tip1Vec.y, z: tip1Vec.z };
        const tip2 = { x: tip2Vec.x, y: tip2Vec.y, z: tip2Vec.z };

        // Position Load midway between hoist points, driven by hoist wire payout
        const loadX = (tip1.x + tip2.x) / 2;
        const avgTipY = (tip1.y + tip2.y) / 2;
        const hookY = Math.max(avgTipY - currentHoistDrop, 4.0);
        const loadY = Math.max(hookY - 2.5, 2.1);
        const loadZ = (tip1.z + tip2.z) / 2;
        suspendedLoad.position.set(loadX, loadY, loadZ);

        // Hooks ride below tips on hoist cables
        hook1Mesh.position.set(tip1.x, hookY, tip1.z);
        hook2Mesh.position.set(tip2.x, hookY, tip2.z);

        // Hoist pendant lines: boom tips down to hooks
        if (hoistLines && hoistLines.length === 2) {
            const hEnds = [
                [tip1, hook1Mesh.position],
                [tip2, hook2Mesh.position]
            ];
            hoistLines.forEach(function (line, i) {
                const p = line.geometry.attributes.position;
                p.setXYZ(0, hEnds[i][0].x, hEnds[i][0].y, hEnds[i][0].z);
                p.setXYZ(1, hEnds[i][1].x, hEnds[i][1].y, hEnds[i][1].z);
                p.needsUpdate = true;
            });
        }

        // Slings connect from hook blocks to load lifting trunnions/shackles
        const ends = [
            [{ x: hook1Mesh.position.x, y: hook1Mesh.position.y - 0.8, z: hook1Mesh.position.z }, { x: loadX - state.load.length / 2 + 1, y: loadY + state.load.radius, z: loadZ }],
            [{ x: hook1Mesh.position.x, y: hook1Mesh.position.y - 0.8, z: hook1Mesh.position.z }, { x: loadX - state.load.length / 4, y: loadY + state.load.radius, z: loadZ }],
            [{ x: hook2Mesh.position.x, y: hook2Mesh.position.y - 0.8, z: hook2Mesh.position.z }, { x: loadX + state.load.length / 4, y: loadY + state.load.radius, z: loadZ }],
            [{ x: hook2Mesh.position.x, y: hook2Mesh.position.y - 0.8, z: hook2Mesh.position.z }, { x: loadX + state.load.length / 2 - 1, y: loadY + state.load.radius, z: loadZ }]
        ];
        slingLines.forEach(function (line, i) {
            const p = line.geometry.attributes.position;
            p.setXYZ(0, ends[i][0].x, ends[i][0].y, ends[i][0].z);
            p.setXYZ(1, ends[i][1].x, ends[i][1].y, ends[i][1].z);
            p.needsUpdate = true;
        });

        live.radius1 = Math.hypot(tip1.x - state.crane1.x, tip1.z - state.crane1.z);
        live.radius2 = Math.hypot(tip2.x - state.crane2.x, tip2.z - state.crane2.z);

        // Evaluate Spatial Distance & Collision
        const dx = tip1.x - tip2.x;
        const dy = tip1.y - tip2.y;
        const dz = tip1.z - tip2.z;
        const dist = Math.sqrt(dx*dx + dy*dy + dz*dz);
        const clearance = dist - state.load.length;

        // Offline mirror of the bearing check (display estimation only):
        // per-crane share + user machine weight over 4 pads.
        // Offline mirror: display-only estimation (tagged). Ratings arrive
        // from the engines; machine weights are user inputs, not constants.
        const padLoad1 = (state.load.massTonnes * 0.52 + state.plan.chassis1) / 4;
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
        live.clearance = clearanceEl.textContent + ' · OFFLINE EST';
        live.fosText = gbpEl.textContent + ' · OFFLINE EST';
        live.fos = fos;
        live.share1 = 52;
        verdictOnline = false;
        refreshInspectorLive();
        scheduleVerdicts();
    }

    function verdictPayload() {
        const P = state.plan;
        const crane = (c, chassis) => ({
            base_position: { x: c.x, y: 0, z: c.z },
            boom_length_meters: c.boomLength,
            boom_angle_deg: c.boomAngle,
            slew_angle_deg: c.slewAngle,
            counterweight_tonne: P.counterweight,
            outrigger_spread_x_m: P.spread,
            outrigger_spread_z_m: P.spread,
            chassis_weight_tonne: chassis
        });
        return {
            crane1: crane(state.crane1, P.chassis1),
            crane2: crane(state.crane2, P.chassis2),
            totalLoadT: state.load.massTonnes,
            cogOffsetM: P.cogOffset,
            matAreaM2: P.matArea,
            allowableKPa: state.allowableGbpKPa,
            crane1CapT: P.cap1,
            crane2CapT: P.cap2,
            slingAngleDeg: P.slingAngle,
            slingMblT: P.slingMBL,
            riggingT: P.rigging,
            pages: 8
        };
    }

    // Server-authoritative verdicts: the browser mirror above is display-only
    // estimation (offline fallback). Ratings arrive from the engines.
    const API_BASE = window.location.port === '18080' ? '' : 'http://127.0.0.1:18080';

    function scheduleVerdicts() {
        if (verdictTimer) clearTimeout(verdictTimer);
        verdictTimer = setTimeout(async () => {
            try {
                const res = await fetch(`${API_BASE}/api/v1/liftviews/export?mode=verdict`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(verdictPayload())
                });
                if (!res.ok) return;
                const m = await res.json();
                const span = m.clearance_m;
                const fosMin = Math.min(m.fos1, m.fos2);
                const clearanceEl = document.getElementById('clearance-display');
                const gbpEl = document.getElementById('gbp-display');
                clearanceEl.textContent = span > 2.0 ? `CLEAR (${span.toFixed(2)}m) · LIVE` : span > 0.5 ? `WARN (${span.toFixed(2)}m) · LIVE` : `COLLISION (${span.toFixed(2)}m) · LIVE`;
                clearanceEl.className = 'val ' + (span > 2.0 ? 'safe' : span > 0.5 ? 'warn' : 'danger');
                gbpEl.textContent = fosMin >= 1.5 ? `PASS (FoS ${fosMin.toFixed(2)}) · LIVE` : `FAIL (FoS ${fosMin.toFixed(2)}) · LIVE`;
                gbpEl.className = 'val ' + (fosMin >= 1.5 ? 'safe' : 'danger');
                live.clearance = clearanceEl.textContent;
                live.fosText = gbpEl.textContent;
                live.fos = fosMin;
                live.share1 = Math.round(m.share1_pct);
                verdictOnline = true;
                refreshInspectorLive();
            } catch (e) {
                // Offline fallback: keep local math active without console spam
                verdictOnline = false;
            }
        }, 400);
    }

    // Realistic 4D Lift Sequence Stages (t: 0 to 60s)
    // Stage 0: Rigging Pre-tension (on transport trailer, cables tension up)
    // Stage 1: Initial Lift-off & Hold (clear trailer bed, verify load share & brakes)
    // Stage 2: Synchronized Vertical Hoist (reach transit clearance height)
    // Stage 3: Synchronized Tandem Slew & Travel (swing towards foundation pad)
    // Stage 4: Controlled Lowering & Landing on Foundation Plinths
    const LIFT_STAGES = [
        { t: 0.0,  name: 'Stage 0: Rigging Pre-tension', status: 'warn',   c1Slew: 36.0, c1Boom: 63.5, c2Slew: -36.0, c2Boom: 63.5, hoistDrop: 16.5 },
        { t: 8.0,  name: 'Stage 0: Rigging Pre-tension', status: 'warn',   c1Slew: 36.0, c1Boom: 63.5, c2Slew: -36.0, c2Boom: 63.5, hoistDrop: 16.5 },
        { t: 15.0, name: 'Stage 1: Lift-off & Brake Check', status: 'safe', c1Slew: 36.0, c1Boom: 63.5, c2Slew: -36.0, c2Boom: 63.5, hoistDrop: 14.5 },
        { t: 28.0, name: 'Stage 2: Vertical Hoist Clearance', status: 'safe', c1Slew: 36.0, c1Boom: 64.0, c2Slew: -36.0, c2Boom: 64.0, hoistDrop: 8.0 },
        { t: 46.0, name: 'Stage 3: Synchronized Tandem Slew', status: 'safe', c1Slew: 12.0, c1Boom: 66.0, c2Slew: -12.0, c2Boom: 66.0, hoistDrop: 8.0 },
        { t: 60.0, name: 'Stage 4: Landing on Foundation', status: 'safe',  c1Slew: 12.0, c1Boom: 66.0, c2Slew: -12.0, c2Boom: 66.0, hoistDrop: 16.2 }
    ];

    function evaluateLiftStage(t) {
        if (t <= LIFT_STAGES[0].t) return { ...LIFT_STAGES[0] };
        const last = LIFT_STAGES[LIFT_STAGES.length - 1];
        if (t >= last.t) return { ...last };

        for (let i = 0; i < LIFT_STAGES.length - 1; i++) {
            const k0 = LIFT_STAGES[i];
            const k1 = LIFT_STAGES[i + 1];
            if (t >= k0.t && t <= k1.t) {
                const fraction = (t - k0.t) / (k1.t - k0.t);
                // Smooth step interpolation for crane movements
                const smooth = fraction * fraction * (3 - 2 * fraction);
                return {
                    name: k1.name,
                    status: k1.status,
                    c1Slew: k0.c1Slew + (k1.c1Slew - k0.c1Slew) * smooth,
                    c1Boom: k0.c1Boom + (k1.c1Boom - k0.c1Boom) * smooth,
                    c2Slew: k0.c2Slew + (k1.c2Slew - k0.c2Slew) * smooth,
                    c2Boom: k0.c2Boom + (k1.c2Boom - k0.c2Boom) * smooth,
                    hoistDrop: k0.hoistDrop + (k1.hoistDrop - k0.hoistDrop) * smooth
                };
            }
        }
        return { ...last };
    }

    let currentHoistDrop = 16.5;

    function setupEventListeners() {
        const sliderTime = document.getElementById('time-slider');
        sliderTime.addEventListener('input', (e) => {
            state.time = parseFloat(e.target.value);
            // Drive discrete, multi-stage realistic lift trajectory
            const stage = evaluateLiftStage(state.time);
            state.crane1.slewAngle = stage.c1Slew;
            state.crane1.boomAngle = stage.c1Boom;
            state.crane2.slewAngle = stage.c2Slew;
            state.crane2.boomAngle = stage.c2Boom;
            currentHoistDrop = stage.hoistDrop;

            // Update UI phase display badge & slider displays
            const phaseEl = document.getElementById('phase-display');
            if (phaseEl) {
                phaseEl.textContent = stage.name;
                phaseEl.className = 'val ' + stage.status;
            }
            const c1AngleEl = document.getElementById('crane1-angle-val');
            const c1SlewEl = document.getElementById('crane1-slew-val');
            const c2AngleEl = document.getElementById('crane2-angle-val');
            const c2SlewEl = document.getElementById('crane2-slew-val');
            if (c1AngleEl) c1AngleEl.textContent = `${stage.c1Boom.toFixed(1)}°`;
            if (c1SlewEl) c1SlewEl.textContent = `${stage.c1Slew.toFixed(1)}°`;
            if (c2AngleEl) c2AngleEl.textContent = `${stage.c2Boom.toFixed(1)}°`;
            if (c2SlewEl) c2SlewEl.textContent = `${stage.c2Slew.toFixed(1)}°`;
            const s1 = document.getElementById('crane1-angle');
            const s2 = document.getElementById('crane1-slew');
            const s3 = document.getElementById('crane2-angle');
            const s4 = document.getElementById('crane2-slew');
            if (s1) s1.value = stage.c1Boom;
            if (s2) s2.value = stage.c1Slew;
            if (s3) s3.value = stage.c2Boom;
            if (s4) s4.value = stage.c2Slew;

            updateKinematics();
        });
        // Kinematic sliders (were dead inputs — now drive state).
        [['crane1-angle', 'crane1', 'boomAngle', 'crane1-angle-val', '°'],
         ['crane1-slew', 'crane1', 'slewAngle', 'crane1-slew-val', '°'],
         ['crane2-angle', 'crane2', 'boomAngle', 'crane2-angle-val', '°'],
         ['crane2-slew', 'crane2', 'slewAngle', 'crane2-slew-val', '°']
        ].forEach(([id, crane, key, label, unit]) => {
            const el = document.getElementById(id);
            el.addEventListener('input', (e) => {
                state[crane][key] = parseFloat(e.target.value);
                document.getElementById(label).textContent = e.target.value + unit;
                updateKinematics();
            });
        });

        // Plan inputs: user data straight into state (engine gates it all).
        const planBind = [
            ['in-load', (v) => { state.load.massTonnes = v; }],
            ['in-cap1', (v) => { state.plan.cap1 = v; }],
            ['in-cap2', (v) => { state.plan.cap2 = v; }],
            ['in-cw', (v) => { state.plan.counterweight = v; }],
            ['in-chassis1', (v) => { state.plan.chassis1 = v; }],
            ['in-chassis2', (v) => { state.plan.chassis2 = v; }],
            ['in-spread', (v) => { state.plan.spread = v; }],
            ['in-sling-angle', (v) => { state.plan.slingAngle = v; }],
            ['in-sling-mbl', (v) => { state.plan.slingMBL = v; }],
            ['in-rigging', (v) => { state.plan.rigging = v; }],
            ['in-mat', (v) => { state.plan.matArea = v; }],
            ['in-allow', (v) => { state.allowableGbpKPa = v; }],
            ['in-cog', (v) => { state.plan.cogOffset = v; }]
        ];
        const geomBind = [
            ['in-boom1', (v) => { state.crane1.boomLength = v; rebuildBoom(boom1, v); }],
            ['in-boom2', (v) => { state.crane2.boomLength = v; rebuildBoom(boom2, v); }],
            ['in-x1', (v) => { state.crane1.x = v; crane1Group.position.x = v; }],
            ['in-x2', (v) => { state.crane2.x = v; crane2Group.position.x = v; }],
            ['in-llen', (v) => { state.load.length = v; rebuildLoad(); }],
            ['in-lrad', (v) => { state.load.radius = v; rebuildLoad(); }],
            ['in-T', (v) => {
                state.window = v;
                document.getElementById('time-slider').max = v;
                if (state.time > v) { state.time = v; document.getElementById('time-slider').value = v; }
            }]
        ];
        planBind.concat(geomBind).forEach(([id, set]) => {
            document.getElementById(id).addEventListener('input', (e) => {
                const v = parseFloat(e.target.value);
                if (!isNaN(v)) { set(v); updateKinematics(); }
            });
        });

        document.getElementById('btn-play').addEventListener('click', () => {
            if (!isPlaying) {
                isPlaying = true;
                document.getElementById('btn-play').textContent = '⏸ Pause';
                playInterval = setInterval(() => {
                    if (state.time < state.window) {
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

        document.getElementById('btn-export').addEventListener('click', exportSolvedPack);

        // Hierarchy selection: click objects, toolbar modes, search, keys.
        const dom = renderer.domElement;
        dom.addEventListener('click', (e) => {
            const r = dom.getBoundingClientRect();
            const mouse = new THREE.Vector2(
                ((e.clientX - r.left) / r.width) * 2 - 1,
                -((e.clientY - r.top) / r.height) * 2 + 1
            );
            const ray = new THREE.Raycaster();
            ray.setFromCamera(mouse, camera);
            const hits = ray.intersectObjects(scene.children, true);
            for (const h of hits) {
                let o = h.object;
                while (o && !o.userData.nodeId) o = o.parent;
                if (o && o.userData.nodeId) {
                    selectNode(o.userData.nodeId);
                    return;
                }
            }
            selectNode(null);
        });
        document.querySelectorAll('#view-toolbar button').forEach((b) => {
            b.addEventListener('click', () => {
                viewMode = b.dataset.mode;
                document.querySelectorAll('#view-toolbar button').forEach((x) => x.classList.remove('active'));
                b.classList.add('active');
                applyViewMode();
            });
        });
        document.getElementById('system-search').addEventListener('input', (e) => {
            buildSystemsList(e.target.value.toLowerCase());
        });
        document.addEventListener('keydown', (e) => {
            if (e.target.tagName === 'INPUT') return;
            const k = e.key.toLowerCase();
            if (k === 'f') setToolbarMode('focus');
            else if (k === 'i') setToolbarMode('isolate');
            else if (k === 'd') setToolbarMode('fade');
            else if (k === 'h') setToolbarMode('hide');
            else if (k === 'u' || k === 'r') setToolbarMode('reset');
            else if (k === 'escape') selectNode(null);
        });
    }

    async function exportSolvedPack() {
        const statusEl = document.getElementById('export-status');
        statusEl.textContent = 'solving…';
        // Raw scene state only: all ratings happen server-side in the engines.
        const payload = verdictPayload();
        try {
            const res = await fetch(`${API_BASE}/api/v1/liftviews/export`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });
            if (!res.ok) {
                statusEl.textContent = 'refused: ' + res.status + ' ' + (await res.text()).slice(0, 160);
                return;
            }
            const blob = await res.blob();
            const a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = 'liftview-export.zip';
            a.click();
            setTimeout(() => URL.revokeObjectURL(a.href), 5000);
            statusEl.textContent = 'downloaded';
        } catch (e) {
            statusEl.textContent = 'error: server unreachable';
        }
    }

    function setToolbarMode(mode) {
        viewMode = mode;
        document.querySelectorAll('#view-toolbar button').forEach((x) => x.classList.toggle('active', x.dataset.mode === mode));
        applyViewMode();
    }

    function taggedMeshes() {
        const out = [];
        scene.traverse((o) => {
            if ((o.isMesh || o.isLine) && o.userData.nodeId) out.push(o);
        });
        return out;
    }

    function applyViewMode() {
        taggedMeshes().forEach((o) => {
            const hasSelection = selectedId !== null;
            const isSel = o.userData.nodeId === selectedId;

            if (o.isLine) {
                o.visible = !hasSelection || viewMode === 'reset' || viewMode === 'fade' || isSel || viewMode === 'focus';
                return;
            }

            o.visible = true;
            const m = o.material;
            if (!m || !('opacity' in m)) return;

            if (!hasSelection || viewMode === 'reset') {
                m.transparent = false;
                m.opacity = 1;
                if ('wireframe' in m) m.wireframe = false;
            } else if (viewMode === 'fade') {
                m.transparent = !isSel;
                m.opacity = isSel ? 1 : 0.2;
            } else if (viewMode === 'hide' || viewMode === 'isolate') {
                o.visible = isSel;
            } else if (viewMode === 'xray') {
                if (o.isMesh && ('wireframe' in m)) {
                    m.transparent = true;
                    m.opacity = isSel ? 1 : 0.35;
                    m.wireframe = !isSel;
                }
            } else if (viewMode === 'focus' && isSel) {
                const p = new THREE.Vector3();
                o.getWorldPosition(p);
                if (controls) controls.target.copy(p);
            }
        });
    }

    function buildSystemsList(filter) {
        const host = document.getElementById('systems-list');
        host.innerHTML = '';
        const H = window.INTEGIN_HIERARCHY.tree;
        let total = 0;
        H.systems.forEach((s) => {
            const comps = s.components.filter((c) =>
                !filter || (c.name + ' ' + c.id + ' ' + c.desc).toLowerCase().includes(filter));
            if (!comps.length) return;
            total += comps.length;
            const div = document.createElement('div');
            div.className = 'system-group';
            div.innerHTML = `<h3>${s.icon} ${s.name} <span class="count">${comps.length}</span></h3>`;
            comps.forEach((c) => {
                const b = document.createElement('button');
                b.className = 'comp-item' + (c.id === selectedId ? ' selected' : '');
                b.textContent = `${c.name} · ${c.id}`;
                b.addEventListener('click', () => selectNode(c.id));
                div.appendChild(b);
            });
            host.appendChild(div);
        });
        document.getElementById('assembly-summary').textContent =
            `${H.systems.length} systems · ${total} components shown`;
    }

    function selectNode(id) {
        selectedId = id;
        applyViewMode();
        buildSystemsList((document.getElementById('system-search').value || '').toLowerCase());
        renderInspector();
        renderBreadcrumb();
    }

    function liveSpecs(id) {
        switch (id) {
            case 'CRN-ALPHA': return [['Working radius', live.radius1.toFixed(2) + ' m'], ['Machine weight', state.plan.chassis1 + ' t'], ['Load share', live.share1 + ' / ' + (100 - live.share1)]];
            case 'CRN-BETA': return [['Working radius', live.radius2.toFixed(2) + ' m'], ['Machine weight', state.plan.chassis2 + ' t'], ['Load share', (100 - live.share1) + ' / ' + live.share1]];
            case 'HOOK-ALPHA': return [['Hook share', (45 * live.share1 / 100).toFixed(1) + ' t']];
            case 'HOOK-BETA': return [['Hook share', (45 * (100 - live.share1) / 100).toFixed(1) + ' t']];
            case 'SLING-SET': return [['Clearance verdict', live.clearance]];
            case 'SHACKLE-SET': return [['Lateral allowance', (45 * 0.03).toFixed(2) + ' t min']];
            case 'VESSEL-45T': return [['DHL share', live.clearance]];
            case 'MAT-SYS': return [['Bearing verdict', live.fosText], ['FoS', live.fos.toFixed(2)]];
            case 'LIFT-SEQ': return [['Clearance', live.clearance], ['Bearing', live.fosText]];
            default: return [];
        }
    }

    function renderInspector() {
        const host = document.getElementById('inspector');
        if (!selectedId) {
            host.innerHTML = '<p class="hint">Select a system or click an object in 3D.</p>';
            return;
        }
        const found = window.INTEGIN_HIERARCHY.findNode(selectedId);
        if (!found) return;
        const { system, comp } = found;
        const tabs = ['Overview', 'Engineering', 'Safety', 'Specs', 'Evidence'];
        let body = '';
        if (inspectorTab === 'Overview') body = `<p>${comp.desc}</p>`;
        else if (inspectorTab === 'Engineering') {
            body = liveSpecs(comp.id).map((r) => `<div class="kv"><span>${r[0]}</span><b>${r[1]}</b></div>`).join('') ||
                '<p class="hint">No live values for this node.</p>';
        }
        else if (inspectorTab === 'Safety') body = `<p>${comp.safety}</p>`;
        else if (inspectorTab === 'Specs') body = comp.specs.map((r) => `<div class="kv"><span>${r[0]}</span><b>${r[1]}</b></div>`).join('');
        else body = `<p>${comp.evidence}</p>`;
        host.innerHTML = `
            <div class="insp-head"><span class="crumb-sys">${system.name}</span>
            <span class="badge">${comp.status}</span></div>
            <h3>${comp.name}</h3>
            <div class="insp-id">${comp.id} · ${comp.level}</div>
            <div class="insp-tabs">${tabs.map((t) => `<button class="${t === inspectorTab ? 'active' : ''}" data-tab="${t}">${t}</button>`).join('')}</div>
            <div class="insp-body">${body}</div>
            <div class="insp-std">${comp.standards.map((s) => `<div>§ <b>${s[0]}</b> — ${s[1]}</div>`).join('')}</div>`;
        host.querySelectorAll('[data-tab]').forEach((b) => {
            b.addEventListener('click', () => { inspectorTab = b.dataset.tab; renderInspector(); });
        });
    }

    function refreshInspectorLive() {
        if (selectedId && inspectorTab === 'Engineering') renderInspector();
    }

    function renderBreadcrumb() {
        const el = document.getElementById('breadcrumb');
        if (!selectedId) { el.textContent = 'Tandem lift · no selection'; return; }
        const found = window.INTEGIN_HIERARCHY.findNode(selectedId);
        if (!found) return;
        el.textContent = `${found.system.name} › ${found.comp.name} · ${viewMode}`;
    }

    function onWindowResize() {
        const container = document.getElementById('canvas-container');
        if (!container || !camera || !renderer) return;
        camera.aspect = container.clientWidth / container.clientHeight;
        camera.updateProjectionMatrix();
        renderer.setSize(container.clientWidth, container.clientHeight);
    }

    function animate() {
        requestAnimationFrame(animate);
        if (controls) controls.update();
        if (renderer && scene && camera) {
            renderer.render(scene, camera);
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
