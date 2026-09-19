// INTEGIN Parametric Dynamic Blocks Library Catalog
// Reusable engineered components for 3D simulation and CAD DXF synthesis.
// Standard references: ASME B30.20 (Below-the-Hook), ASME B30.26 (Rigging),
// ASME B30.5 (Mobile Cranes), AISC 360, DNV-ST-N001, EN 13155.

(function () {
    'use strict';

    const BLOCKS_CATALOG = {
        loads: [
            {
                id: 'BLOCK_LOAD_VESSEL',
                category: 'loads',
                name: 'Horizontal Pressure Vessel',
                code: 'PV-01',
                desc: 'Standard cylindrical pressure vessel with dual dished ellipsoidal heads and welded saddle/trunnions.',
                standards: 'ASME Sec VIII Div 1 · WLL 45 t · ASME B30.20',
                defaultProps: {
                    massTonnes: 45.0,
                    length: 12.0,
                    radius: 1.5,
                    color: 0x38bdf8,
                    cogOffset: 0.0
                },
                cadBlock: 'VESSEL_HORIZ_DYN',
                build3D: function (props) {
                    const group = new THREE.Group();
                    // Main cylindrical shell
                    const bodyGeo = new THREE.CylinderGeometry(props.radius, props.radius, props.length, 24);
                    bodyGeo.rotateZ(Math.PI / 2);
                    const mat = new THREE.MeshStandardMaterial({
                        color: props.color || 0x38bdf8,
                        roughness: 0.4,
                        metalness: 0.3
                    });
                    const shell = new THREE.Mesh(bodyGeo, mat);
                    group.add(shell);

                    // Dual Dished Heads (Hemispherical ends)
                    const headMat = mat.clone();
                    [-props.length / 2, props.length / 2].forEach((xPos, idx) => {
                        const headGeo = new THREE.SphereGeometry(props.radius, 16, 16, 0, Math.PI * 2, 0, Math.PI / 2);
                        headGeo.rotateZ(idx === 0 ? Math.PI / 2 : -Math.PI / 2);
                        const head = new THREE.Mesh(headGeo, headMat);
                        head.position.set(xPos, 0, 0);
                        group.add(head);
                    });

                    // Lifting Trunnions (Lug studs)
                    const trunnionGeo = new THREE.CylinderGeometry(0.3, 0.3, props.radius * 2 + 0.8, 12);
                    trunnionGeo.rotateX(Math.PI / 2);
                    const trunnionMat = new THREE.MeshStandardMaterial({ color: 0xf59e0b, metalness: 0.6 });
                    [-props.length / 3, props.length / 3].forEach((xPos) => {
                        const trun = new THREE.Mesh(trunnionGeo, trunnionMat);
                        trun.position.set(xPos, 0, 0);
                        group.add(trun);
                    });

                    // Nozzles on top
                    const nozzleGeo = new THREE.CylinderGeometry(0.25, 0.25, 0.8, 12);
                    const nozzleMat = new THREE.MeshStandardMaterial({ color: 0x64748b });
                    [-2, 0, 2].forEach(nx => {
                        const noz = new THREE.Mesh(nozzleGeo, nozzleMat);
                        noz.position.set(nx, props.radius + 0.4, 0);
                        group.add(noz);
                    });

                    return group;
                }
            },
            {
                id: 'BLOCK_LOAD_COLUMN',
                category: 'loads',
                name: 'Fractionation Column / Demethanizer',
                code: 'COL-02',
                desc: 'Tall vertical process column with top manway, skirt ring base, and asymmetric platform weight.',
                standards: 'ASME Sec VIII / API 510 · ASME B30.20 Tailing Certified',
                defaultProps: {
                    massTonnes: 65.0,
                    length: 22.0,
                    radius: 1.2,
                    color: 0x0ea5e9,
                    cogOffset: -1.5
                },
                cadBlock: 'COLUMN_VERT_DYN',
                build3D: function (props) {
                    const group = new THREE.Group();
                    // Slender cylinder body
                    const colGeo = new THREE.CylinderGeometry(props.radius, props.radius, props.length, 24);
                    colGeo.rotateZ(Math.PI / 2);
                    const mat = new THREE.MeshStandardMaterial({ color: props.color || 0x0ea5e9, roughness: 0.3 });
                    group.add(new THREE.Mesh(colGeo, mat));

                    // Base Skirt (wider diameter flange at bottom / -X side)
                    const skirtGeo = new THREE.CylinderGeometry(props.radius * 1.3, props.radius * 1.3, 2.5, 24);
                    skirtGeo.rotateZ(Math.PI / 2);
                    const skirtMat = new THREE.MeshStandardMaterial({ color: 0x475569, metalness: 0.5 });
                    const skirt = new THREE.Mesh(skirtGeo, skirtMat);
                    skirt.position.set(-props.length / 2 + 1.25, 0, 0);
                    group.add(skirt);

                    // Circular ring platforms (cage walkways)
                    const ringGeo = new THREE.TorusGeometry(props.radius + 0.6, 0.1, 8, 24);
                    ringGeo.rotateY(Math.PI / 2);
                    const ringMat = new THREE.MeshStandardMaterial({ color: 0xfacc15 });
                    [-4, 0, 5].forEach(rx => {
                        const ring = new THREE.Mesh(ringGeo, ringMat);
                        ring.position.set(rx, 0, 0);
                        group.add(ring);
                    });

                    // Top Lifting Trunnions (for primary lead crane)
                    const topTrunGeo = new THREE.CylinderGeometry(0.35, 0.35, props.radius * 2 + 1.0, 12);
                    topTrunGeo.rotateX(Math.PI / 2);
                    const topTrun = new THREE.Mesh(topTrunGeo, new THREE.MeshStandardMaterial({ color: 0xef4444 }));
                    topTrun.position.set(props.length / 2 - 1.5, 0, 0);
                    group.add(topTrun);

                    // Bottom Tailing Lug (for tail crane)
                    const tailLugGeo = new THREE.BoxGeometry(1.2, 0.8, 0.4);
                    const tailLug = new THREE.Mesh(tailLugGeo, new THREE.MeshStandardMaterial({ color: 0xef4444 }));
                    tailLug.position.set(-props.length / 2 + 0.5, 0, props.radius);
                    group.add(tailLug);

                    return group;
                }
            },
            {
                id: 'BLOCK_LOAD_GIRDER',
                category: 'loads',
                name: 'Precast Concrete Bridge Girder',
                code: 'GIR-03',
                desc: 'Prestressed concrete I-girder / box girder for civil viaduct erection with embedded cast-in lift loops.',
                standards: 'AASHTO LRFD Bridge Spec · PCI Design Handbook · FoS ≥ 3.0',
                defaultProps: {
                    massTonnes: 38.0,
                    length: 18.0,
                    radius: 1.1,
                    color: 0x94a3b8,
                    cogOffset: 0.0
                },
                cadBlock: 'GIRDER_PRECAST_DYN',
                build3D: function (props) {
                    const group = new THREE.Group();
                    const girderMat = new THREE.MeshStandardMaterial({ color: 0x94a3b8, roughness: 0.9 });

                    // Web (center box)
                    const webGeo = new THREE.BoxGeometry(props.length, 1.8, 0.5);
                    group.add(new THREE.Mesh(webGeo, girderMat));

                    // Top Flange
                    const topFlangeGeo = new THREE.BoxGeometry(props.length, 0.4, 1.6);
                    const topFlange = new THREE.Mesh(topFlangeGeo, girderMat);
                    topFlange.position.y = 1.0;
                    group.add(topFlange);

                    // Bottom Flange
                    const btmFlangeGeo = new THREE.BoxGeometry(props.length, 0.5, 2.0);
                    const btmFlange = new THREE.Mesh(btmFlangeGeo, girderMat);
                    btmFlange.position.y = -1.0;
                    group.add(btmFlange);

                    // Cast-in lift loops (padeyes)
                    const loopMat = new THREE.MeshStandardMaterial({ color: 0xf59e0b, metalness: 0.8 });
                    const loopGeo = new THREE.TorusGeometry(0.25, 0.08, 8, 16);
                    [-props.length / 2 + 1.5, props.length / 2 - 1.5].forEach(lx => {
                        const loop = new THREE.Mesh(loopGeo, loopMat);
                        loop.position.set(lx, 1.35, 0);
                        group.add(loop);
                    });

                    return group;
                }
            },
            {
                id: 'BLOCK_LOAD_MODULE',
                category: 'loads',
                name: 'Modular Substation / E-House Skid',
                code: 'SKID-04',
                desc: 'Structural containerized electrical control skid with 4-corner ISO/pad-eye lift points for multi-crane rigging.',
                standards: 'DNVGL-ST-E271 / ISO 1496 · 4-Point Rigged',
                defaultProps: {
                    massTonnes: 52.0,
                    length: 14.0,
                    radius: 1.8,
                    color: 0x10b981,
                    cogOffset: 0.3
                },
                cadBlock: 'EHOUSE_SKID_DYN',
                build3D: function (props) {
                    const group = new THREE.Group();
                    // Main architectural enclosure
                    const bodyGeo = new THREE.BoxGeometry(props.length, 3.2, 3.4);
                    const bodyMat = new THREE.MeshStandardMaterial({ color: 0x1e3a5f, roughness: 0.6 });
                    group.add(new THREE.Mesh(bodyGeo, bodyMat));

                    // Corrugated structural frame corners
                    const frameMat = new THREE.MeshStandardMaterial({ color: 0x10b981, metalness: 0.5 });
                    const cornerGeo = new THREE.BoxGeometry(0.3, 3.3, 0.3);
                    [
                        [-props.length / 2, 0, -1.7],
                        [-props.length / 2, 0, 1.7],
                        [props.length / 2, 0, -1.7],
                        [props.length / 2, 0, 1.7]
                    ].forEach(pos => {
                        const c = new THREE.Mesh(cornerGeo, frameMat);
                        c.position.set(pos[0], pos[1], pos[2]);
                        group.add(c);
                    });

                    // Roof HVAC units
                    const hvacGeo = new THREE.BoxGeometry(2.0, 0.8, 1.8);
                    const hvacMat = new THREE.MeshStandardMaterial({ color: 0x64748b });
                    [-3, 3].forEach(hx => {
                        const hvac = new THREE.Mesh(hvacGeo, hvacMat);
                        hvac.position.set(hx, 2.0, 0);
                        group.add(hvac);
                    });

                    // 4-Corner Top Padeyes
                    const eyeMat = new THREE.MeshStandardMaterial({ color: 0xf59e0b, metalness: 0.8 });
                    const eyeGeo = new THREE.TorusGeometry(0.2, 0.07, 8, 16);
                    [
                        [-props.length / 2 + 0.4, 1.8, -1.5],
                        [-props.length / 2 + 0.4, 1.8, 1.5],
                        [props.length / 2 - 0.4, 1.8, -1.5],
                        [props.length / 2 - 0.4, 1.8, 1.5]
                    ].forEach(ep => {
                        const eye = new THREE.Mesh(eyeGeo, eyeMat);
                        eye.position.set(ep[0], ep[1], ep[2]);
                        group.add(eye);
                    });

                    return group;
                }
            },
            {
                id: 'BLOCK_LOAD_TRANSFORMER',
                category: 'loads',
                name: 'Heavy Power Transformer Skid',
                code: 'XFRM-05',
                desc: 'Main step-up power transformer with high voltage bushings, conservator tank, and radiator bank.',
                standards: 'IEEE C57 / IEC 60076 · Concentrated Heavy Rigging',
                defaultProps: {
                    massTonnes: 78.0,
                    length: 9.0,
                    radius: 2.0,
                    color: 0x8b5cf6,
                    cogOffset: -0.6
                },
                cadBlock: 'TRANSFORMER_DYN',
                build3D: function (props) {
                    const group = new THREE.Group();
                    // Main Tank Core
                    const coreGeo = new THREE.BoxGeometry(6.5, 3.8, 3.2);
                    const coreMat = new THREE.MeshStandardMaterial({ color: 0x475569, roughness: 0.5 });
                    group.add(new THREE.Mesh(coreGeo, coreMat));

                    // Conservator Tank on top
                    const consGeo = new THREE.CylinderGeometry(0.7, 0.7, 5.0, 16);
                    consGeo.rotateZ(Math.PI / 2);
                    const consMat = new THREE.MeshStandardMaterial({ color: 0x8b5cf6 });
                    const cons = new THREE.Mesh(consGeo, consMat);
                    cons.position.set(0, 2.6, -1.0);
                    group.add(cons);

                    // Radiator Cooling Fins (Flanking side)
                    const radGeo = new THREE.BoxGeometry(5.8, 2.8, 0.6);
                    const radMat = new THREE.MeshStandardMaterial({ color: 0x334155 });
                    const rad1 = new THREE.Mesh(radGeo, radMat);
                    rad1.position.set(0, 0, 1.8);
                    group.add(rad1);

                    // 3 High-voltage Bushings (porcelain cones)
                    [-1.8, 0, 1.8].forEach(bx => {
                        const bGeo = new THREE.CylinderGeometry(0.12, 0.28, 1.5, 12);
                        const bMat = new THREE.MeshStandardMaterial({ color: 0x93c5fd });
                        const bush = new THREE.Mesh(bGeo, bMat);
                        bush.position.set(bx, 2.7, 0.6);
                        group.add(bush);
                    });

                    // 4 Heavy Lift Trunnions on sides
                    const trunMat = new THREE.MeshStandardMaterial({ color: 0xef4444, metalness: 0.7 });
                    const trunGeo = new THREE.CylinderGeometry(0.3, 0.3, 4.2, 12);
                    trunGeo.rotateX(Math.PI / 2);
                    [-2.2, 2.2].forEach(tx => {
                        const tr = new THREE.Mesh(trunGeo, trunMat);
                        tr.position.set(tx, 0.8, 0);
                        group.add(tr);
                    });

                    return group;
                }
            }
        ],

        rigging: [
            {
                id: 'BLOCK_RIG_SPREADER_10M',
                category: 'rigging',
                name: 'Modular Spreader Beam (10m / 50t WLL)',
                code: 'SPRD-10M',
                desc: 'End-lug tubular spreader bar for two-point or four-point equalized load distribution.',
                standards: 'ASME B30.20 Below-The-Hook Design Category B · Service Class 2 · WLL 50 t',
                defaultProps: { length: 10.0, diameter: 0.5, wllTonnes: 50.0, tareWeightTonnes: 1.8 },
                cadBlock: 'SPREADER_BEAM_DYN'
            },
            {
                id: 'BLOCK_RIG_SPREADER_16M',
                category: 'rigging',
                name: 'Heavy Spreader Beam (16m / 100t WLL)',
                code: 'SPRD-16M',
                desc: 'Telescopic / flanged modular heavy spreader with multi-hole bottom adjustment lugs.',
                standards: 'DNV-ST-N001 · ASME B30.20 · WLL 100 t · Proof Tested 1.5x',
                defaultProps: { length: 16.0, diameter: 0.7, wllTonnes: 100.0, tareWeightTonnes: 3.4 },
                cadBlock: 'SPREADER_BEAM_HEAVY'
            },
            {
                id: 'BLOCK_RIG_EQUALIZER_TRI',
                category: 'rigging',
                name: 'Equalizer Triangle Plate (75t WLL)',
                code: 'EQ-TRI-75',
                desc: 'Heavy-plate triangular balancing rigging link to distribute load 50/50 between two slings.',
                standards: 'ASME B30.26 Shackles & Rigging Hardware · WLL 75 t',
                defaultProps: { width: 1.2, height: 1.0, thickness: 0.1, wllTonnes: 75.0 },
                cadBlock: 'TRIANGLE_PLATE_DYN'
            },
            {
                id: 'BLOCK_RIG_SNATCH_BLOCK',
                category: 'rigging',
                name: 'Heavy Snatch Block / Traveling Block',
                code: 'BLK-SNATCH',
                desc: 'Wire rope snatch block with bronze bushing and safety locking hook for reeving multiplier.',
                standards: 'Crosby Western / McKissick Spec · ASME B30.26',
                defaultProps: { sheaveDia: 0.6, wireDia: 32, wllTonnes: 40.0 },
                cadBlock: 'SNATCH_BLOCK_DYN'
            }
        ],

        support: [
            {
                id: 'BLOCK_SUP_OUTRIGGER_MAT_3M',
                category: 'support',
                name: 'Heavy Steel Outrigger Mat (3.0 m²)',
                code: 'MAT-3M',
                desc: 'Rigid structural steel hollow section mat pad distributing point jack reactions into subsoil.',
                standards: 'ASME B30.5 §5-1.5 · CIRIA C703 Crane Mat Design',
                defaultProps: { area: 3.0, width: 1.73, length: 1.73, depth: 0.3, allowableKPa: 220.0 },
                cadBlock: 'STEEL_MAT_3M_DYN'
            },
            {
                id: 'BLOCK_SUP_OUTRIGGER_MAT_5M',
                category: 'support',
                name: 'Heavy Timber Outrigger Mat (5.0 m²)',
                code: 'MAT-5M',
                desc: 'Oak / Ekki hardwood interlocking timber spreader mat for low allowable soil bearing.',
                standards: 'CIRIA C703 · FEM 5.004 · Allowable 140 kPa',
                defaultProps: { area: 5.0, width: 2.24, length: 2.24, depth: 0.35, allowableKPa: 140.0 },
                cadBlock: 'TIMBER_MAT_5M_DYN'
            },
            {
                id: 'BLOCK_SUP_TRAILER_SPMT',
                category: 'support',
                name: 'SPMT Staging Bed (6-Axle)',
                code: 'TRL-SPMT',
                desc: 'Self-propelled modular transporter hydraulic deck at initial staging pickup position.',
                standards: 'Goldhofer / Scheuerle SPMT Payload Standards',
                defaultProps: { length: 14.0, width: 3.5, height: 0.6, payloadTonnes: 120.0 },
                cadBlock: 'SPMT_TRAILER_DYN'
            },
            {
                id: 'BLOCK_SUP_FOUNDATION_SADDLE',
                category: 'support',
                name: 'Concrete Pedestal Saddles',
                code: 'FND-SADDLE',
                desc: 'Permanent reinforced concrete vessel saddles with anchor bolt patterns at final setting zone.',
                standards: 'ACI 318 / AISC 360 · Grout & Anchor Bolt Ready',
                defaultProps: { span: 9.0, width: 2.0, height: 1.2, depth: 3.2 },
                cadBlock: 'FOUNDATION_SADDLES_DYN'
            }
        ]
    };

    function getCatalog() {
        return BLOCKS_CATALOG;
    }

    function getBlockById(id) {
        for (const cat of ['loads', 'rigging', 'support']) {
            const found = BLOCKS_CATALOG[cat].find(b => b.id === id);
            if (found) return found;
        }
        return null;
    }

    window.INTEGIN_BLOCKS = {
        catalog: BLOCKS_CATALOG,
        getCatalog: getCatalog,
        getBlockById: getBlockById
    };
})();
