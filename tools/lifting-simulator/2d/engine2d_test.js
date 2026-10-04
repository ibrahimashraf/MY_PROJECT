const assert = require('assert');
const { Vector2D, LIFT_PHYSICS } = require('./engine2d.js');
const { CRANE_BLOCKS_CATALOG } = require('./blocks_catalog.js');

console.log('Running 2D Parametric Dynamic Blocks Kinematic & Physics Tests...');

// 1. Vector2D operations
{
    const v1 = new Vector2D(3, 4);
    assert.strictEqual(v1.length(), 5.0, 'Vector length must be 5');
    const v2 = new Vector2D(1, 1);
    const added = v1.add(v2);
    assert.strictEqual(added.x, 4);
    assert.strictEqual(added.y, 5);
    const dist = v1.dist(new Vector2D(0, 0));
    assert.strictEqual(dist, 5.0);
    console.log('✔ Vector2D math operations passed');
}

// 2. Kinematics Round-Trip
{
    const boomLen = 40.0;
    const pivotX = -2.0;
    const testAngles = [30.0, 45.0, 60.0, 75.0, 85.0];

    testAngles.forEach(alpha => {
        const radius = LIFT_PHYSICS.boomAngleToRadius(alpha, boomLen, pivotX);
        const derivedAlpha = LIFT_PHYSICS.radiusToBoomAngle(radius, boomLen, pivotX);
        const diff = Math.abs(derivedAlpha - alpha);
        assert(diff < 0.001, `Kinematic round-trip mismatch at angle ${alpha}: got ${derivedAlpha}`);
    });
    console.log('✔ Forward and inverse kinematics round-trip passed');
}

// 3. Sling Tension & Critical Angle Derating Gate
{
    const grossLoad = 50.0;
    // 60 degrees: safe
    const res60 = LIFT_PHYSICS.calculateSlingTension(grossLoad, 60.0, 1.15);
    const expected60 = (50.0 / (2.0 * Math.sin((60.0 * Math.PI) / 180.0))) * 1.15;
    assert.strictEqual(res60.legTensionT, Math.round(expected60 * 100) / 100);
    assert.strictEqual(res60.criticalDerated, false);
    assert.strictEqual(res60.status, 'SAFE');

    // 25 degrees: critical angle lockout
    const res25 = LIFT_PHYSICS.calculateSlingTension(grossLoad, 25.0, 1.15);
    assert.strictEqual(res25.criticalDerated, true);
    assert.strictEqual(res25.status, 'CRITICAL_LOCKOUT');
    console.log('✔ Sling tension & critical angle lockout (<30°) passed');
}

// 4. Outrigger Equilibrium & Ground Bearing Pressure
{
    const crane = CRANE_BLOCKS_CATALOG['LIEBHERR_LTM_1500'];
    assert(crane, 'Liebherr LTM 1500 block must exist in catalog');

    const craneState = {
        workingRadiusM: 14.0,
        slewAngleDeg: 0.0,
        outriggerExt: '100%',
        counterweightT: 90.0,
        matAreaM2: 4.0
    };
    const grossLoadT = 45.0;
    const res = LIFT_PHYSICS.calculateOutriggers(crane, craneState, grossLoadT, 350.0);

    const totalWeight = crane.chassis.weightT + craneState.counterweightT + grossLoadT;
    const sumReactions = res.reactionsT.FL + res.reactionsT.FR + res.reactionsT.RR + res.reactionsT.RL;
    const weightDiff = Math.abs(sumReactions - totalWeight);
    assert(weightDiff < 1.0, `Outrigger equilibrium failed: sum=${sumReactions} totalWeight=${totalWeight}`);

    assert(res.groundPressureKP > 0, 'Ground pressure must be positive');
    assert(res.fos >= 1.5, 'Factor of safety must be >= 1.5 for 350 kPa soil');
    assert.strictEqual(res.status, 'PASS', 'FoS must pass for firm ground');

    // Test warning threshold on weaker 220 kPa soil
    const resWarn = LIFT_PHYSICS.calculateOutriggers(crane, craneState, grossLoadT, 220.0);
    assert.strictEqual(resWarn.status, 'WARNING', 'FoS between 1.0 and 1.5 must trigger WARNING');
    console.log('✔ Outrigger moment equilibrium & GBP calculation passed');
}

// 5. Load Chart Step-Down Lookup
{
    const crane = CRANE_BLOCKS_CATALOG['LIEBHERR_LTM_1100'];
    const capAt4m = LIFT_PHYSICS.getRatedCapacity(crane, 33.8, 4.0);
    assert.strictEqual(capAt4m, 75.0, 'Capacity at 4m must be 75t');

    const capAt10m = LIFT_PHYSICS.getRatedCapacity(crane, 33.8, 10.0);
    assert.strictEqual(capAt10m, 26.5, 'Step-down capacity at 10m must be 26.5t (bracketed by 12m)');

    const capOutOfBounds = LIFT_PHYSICS.getRatedCapacity(crane, 33.8, 50.0);
    assert.strictEqual(capOutOfBounds, 0.0, 'Capacity beyond chart must be 0t');
    console.log('✔ Step-down OEM load chart capacity lookup passed');
}

// 6. Complete Catalog OEM Verification
{
    const expectedOEMs = ['LIEBHERR_LTM_1500', 'LIEBHERR_LTM_1100', 'TADANO_ATF_400G', 'TADANO_ATF_200G_5', 'DEMAG_AC_200_1', 'KATO_CR_250', 'MANITOWOC_MLC300'];
    expectedOEMs.forEach(id => {
        const blk = CRANE_BLOCKS_CATALOG[id];
        assert(blk, `OEM block ${id} must exist`);
        assert(blk.chassis.lengthM > 0, `${id} must have chassis length`);
        assert(blk.outriggers.spanXM > 0, `${id} must have outrigger span X`);
        assert(blk.boom.sections.length > 0, `${id} must have boom sections`);
        assert(blk.chart.length > 0, `${id} must have load chart`);
    });
    console.log('✔ All 7 major OEM crane blocks validated successfully');
}

console.log('All 2D Parametric Dynamic Blocks tests PASSED! ✅');
