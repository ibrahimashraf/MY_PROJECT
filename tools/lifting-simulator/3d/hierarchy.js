// INTEGIN lift assembly hierarchy: the engineering tree the navigator and
// inspector follow (never the raw scene graph). Node IDs bind 1:1 to named
// THREE objects via userData.nodeId. Standards cite the corpus authorities.
(function () {
    'use strict';

    const HIERARCHY = {
        systems: [
            {
                id: 'SYS-HOIST', code: 'HS', name: 'Hoisting', icon: '🏗️',
                components: [
                    {
                        id: 'CRN-ALPHA', level: 'Level 2 · assembly', name: 'Crane Alpha (LTM 1500)',
                        status: 'Active', desc: 'Primary mobile crane. Boom tip position and working radius are solved live from boom/slew state.',
                        specs: [['Boom length', 'user-set'], ['Boom angle', 'live'], ['Slew angle', 'live'], ['Counterweight', '20 t'], ['Chassis', '60 t']],
                        standards: [['ASME B30.5', 'Mobile crane capacity and setup'], ['rulesengine/crane_lmi.go', 'Moment utilization gate ≤ 90%']],
                        safety: 'Anti-two-block monitored. Stop-work on LMI trigger.',
                        evidence: 'OEM load chart (owner-supplied, bound at plan build).'
                    },
                    {
                        id: 'CRN-BETA', level: 'Level 2 · assembly', name: 'Crane Beta (ATF 400G)',
                        status: 'Active', desc: 'Secondary mobile crane. Shares the vessel load by lever rule over hook span.',
                        specs: [['Boom length', 'user-set'], ['Boom angle', 'live'], ['Slew angle', 'live'], ['Counterweight', '20 t'], ['Chassis', '60 t']],
                        standards: [['ASME B30.5', 'Mobile crane capacity and setup'], ['rulesengine/crane_lmi.go', 'Moment utilization gate ≤ 90%']],
                        safety: 'Anti-two-block monitored. Stop-work on LMI trigger.',
                        evidence: 'OEM load chart (owner-supplied, bound at plan build).'
                    },
                    {
                        id: 'HOOK-ALPHA', level: 'Level 3 · component', name: 'Hook Block Alpha',
                        status: 'Proposed', desc: 'Hook block and falls for Crane Alpha. Self-weight enters the DHL ledger.',
                        specs: [['Self-weight', '0.6 t (ledger)'], ['Falls', '2'], ['Hook share', 'live']],
                        standards: [['DNV-ST-N001 §5.3', 'SKL ≥ 1.25 on multi-sling legs'], ['rulesengine/liftplan.go', 'DHL = W + Wrig']],
                        safety: 'Hook latch verified before lift-off. 3% lateral allowance in shackle bow.',
                        evidence: 'Hook cert + WLL marking photograph.'
                    },
                    {
                        id: 'HOOK-BETA', level: 'Level 3 · component', name: 'Hook Block Beta',
                        status: 'Proposed', desc: 'Hook block and falls for Crane Beta. Self-weight enters the DHL ledger.',
                        specs: [['Self-weight', '0.6 t (ledger)'], ['Falls', '2'], ['Hook share', 'live']],
                        standards: [['DNV-ST-N001 §5.3', 'SKL ≥ 1.25 on multi-sling legs'], ['rulesengine/liftplan.go', 'DHL = W + Wrig']],
                        safety: 'Hook latch verified before lift-off. 3% lateral allowance in shackle bow.',
                        evidence: 'Hook cert + WLL marking photograph.'
                    }
                ]
            },
            {
                id: 'SYS-RIG', code: 'RG', name: 'Rigging', icon: '🔗',
                components: [
                    {
                        id: 'SLING-SET', level: 'Level 3 · component', name: 'Sling Set (2-leg)',
                        status: 'Proposed', desc: 'Two-leg bridle. Per-leg tension T = Load / (2·sin φ); refused below 30° from horizontal.',
                        specs: [['Legs', '2'], ['Critical angle', '30° from horizontal'], ['Tension/leg', 'live']],
                        standards: [['rulesengine/rigging.go', 'SlingTension2Leg + critical-angle lockout'], ['DNV Marine Ops 1996 Pt.2 Ch.5', 'MBL checks, both criteria fulfilled']],
                        safety: 'Sling angle lockout at 30°. MBL check on every leg.',
                        evidence: 'Sling certs + pre-use inspection record.'
                    },
                    {
                        id: 'SHACKLE-SET', level: 'Level 3 · component', name: 'Shackle Set',
                        status: 'Proposed', desc: 'Bow shackles at load and hook ends. Minimum 3% lateral load in the bow.',
                        specs: [['Lateral allowance', '3% of design load'], ['Lateral load', 'live']],
                        standards: [['rulesengine/liftplan.go', 'ShackleLateralLoad + 3% lateral expression'], ['DNV Marine Ops 1996', 'Shackle MBL criteria']],
                        safety: 'No side loading beyond the 3% allowance. Pins moused.',
                        evidence: 'Shackle certs + WLL markings.'
                    }
                ]
            },
            {
                id: 'SYS-LOAD', code: 'LD', name: 'Load', icon: '🛢️',
                components: [
                    {
                        id: 'VESSEL-45T', level: 'Level 2 · assembly', name: 'Process Vessel (45 t)',
                        status: 'Active', desc: 'Shared lifted object. Mass splits by lever rule over hook span; CoG defaults to span centre.',
                        specs: [['Mass', '45.0 t'], ['Length', '12.0 m'], ['Radius', '1.5 m'], ['DHL share', 'live']],
                        standards: [['DNV RP H103 Ch.9', 'Light/heavy lift taxonomy; DAF discipline'], ['rulesengine/liftplan.go', 'DHL ledger + DAF lookup']],
                        safety: 'Tag lines fitted. No personnel under suspended load.',
                        evidence: 'Vendor weight report + CoG drawing (registered evidence).'
                    }
                ]
            },
            {
                id: 'SYS-GROUND', code: 'GD', name: 'Ground & Site', icon: '🧱',
                components: [
                    {
                        id: 'MAT-SYS', level: 'Level 3 · component', name: 'Outrigger Mats (3.0 m²)',
                        status: 'Active', desc: 'Bearing mats under all 8 pads. Pressure gated against the allowable with FoS readout.',
                        specs: [['Mat area', '3.0 m²'], ['Allowable', '220 kPa'], ['FoS', 'live']],
                        standards: [['rulesengine/geotech.go', 'CheckBearingPressure + FoS verdict'], ['Lorry-loader .xls precedent', 'Independent 1 m² cross-validation']],
                        safety: 'Mats on firm level ground. Re-check after rain.',
                        evidence: 'Geotech allowable + mat certs.'
                    },
                    {
                        id: 'SITE-EQUIP', level: 'Level 3 · component', name: 'Transport Staging Trailer',
                        status: 'Active', desc: 'Heavy transport low-bed trailer supporting vessel at pick point (Stage 0).',
                        specs: [['Deck height', '0.6 m'], ['Axles', 'Multi-axle SPMT/Hydraulic'], ['Payload rating', '60 t']],
                        standards: [['ASME B30.5 §5-3.2', 'Initial rigging clearance & unseating criteria']],
                        safety: 'Chocks deployed, air brakes locked before line tensioning.',
                        evidence: 'Trailer inspection tag & load positioning survey.'
                    },
                    {
                        id: 'SITE-FOUNDATION', level: 'Level 3 · component', name: 'Target Foundation Plinths',
                        status: 'Active', desc: 'Reinforced concrete pedestals / saddles at final placement coordinates.',
                        specs: [['Elevation', '0.6 m'], ['Span', '9.0 m c-to-c'], ['Anchor bolts', '4x M36 ea']],
                        standards: [['AISC 360 / ACI 318', 'Equipment pedestal setting tolerances']],
                        safety: 'Guide pins fitted, perimeter barricades active.',
                        evidence: 'Civil foundation release certificate.'
                    }
                ]
            },
            {
                id: 'SYS-OPS', code: 'OP', name: 'Operation', icon: '📋',
                components: [
                    {
                        id: 'LIFT-SEQ', level: 'Level 1 · operation', name: 'Tandem Lift Sequence (t 0–60 s)',
                        status: 'Planned', desc: '60-second 4D lift window. Slew sweeps +20° Alpha / +25° Beta; clearance and FoS evaluated per step.',
                        specs: [['Window', '60 s'], ['Clearance verdict', 'live'], ['Bearing verdict', 'live']],
                        standards: [['rulesengine', 'All physics gates; zero JS-side rating math (target)'], ['Plan §8', 'Execution record seals the lift']],
                        safety: 'Stop-work authority with any party. Abort on WARN→COLLISION transition.',
                        evidence: 'Sealed execution record + Merkle log entry.'
                    }
                ]
            }
        ]
    };

    function findNode(id) {
        for (const s of HIERARCHY.systems) {
            for (const c of s.components) {
                if (c.id === id) return { system: s, comp: c };
            }
        }
        return null;
    }

    window.INTEGIN_HIERARCHY = { tree: HIERARCHY, findNode: findNode };
})();


