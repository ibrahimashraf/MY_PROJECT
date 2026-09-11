/* cranes.js
 * Demo crane geometry data for the 2D parametric lifting simulator.
 *
 * DANGER: These are NOT engineering data. All dimensions are illustrative
 * demo values chosen to look plausible. This tool and its data are a PLANNING
 * AID ONLY and are NOT rated, certified, or fit for any lifting decision.
 * Certified capacity/radius values (CENPs) come exclusively from the CEL
 * nanocell process. Never use these numbers for real lifting work.
 */

(function (root, factory) {
    if (typeof module === 'object' && module.exports) {
        module.exports = factory();
    } else {
        root.CRANES = factory();
    }
})(typeof self !== "undefined" ? self : this, function () {
    return {
        liebherr_ltm1160: {
            id: "liebherr_ltm1160",
            name: "Liebherr LTM 1160-5.2 (demo)",
            brand: "Liebherr",
            model: "LTM 1160-5.2",
            // DEMO dimensions — not rated. All lengths in metres, angles in degrees.
            carriageHeight: 3.0,   // deck height above ground
            pivotHeight: 3.2,      // boom pivot above deck
            baseSwingRadius: 2.5,  // pivot offset from turret rotation axis
            baseBoomLen: 63.0,     // main boom length (fully telescoped in this demo pose)
            baseJibLen: 14.0,      // fitted lattice jib
            minBoomAngle: 3,       // minimum boom elevation above horizontal
            maxBoomAngle: 83,      // maximum boom elevation
            minJibAngle: 0,        // jib offset angle relative to boom axis
            maxJibAngle: 40,
            counterweightRadius: 5.9
        },
        tadano_gr1000: {
            id: "tadano_gr1000",
            name: "Tadano GR-1000XL (demo)",
            brand: "Tadano",
            model: "GR-1000XL",
            carriageHeight: 2.9,
            pivotHeight: 3.0,
            baseSwingRadius: 2.3,
            baseBoomLen: 62.5,
            baseJibLen: 10.0,
            minBoomAngle: 2,
            maxBoomAngle: 82,
            minJibAngle: 0,
            maxJibAngle: 30,
            counterweightRadius: 5.5
        },
        kato_nk750: {
            id: "kato_nk750",
            name: "Kato NK-750V (demo)",
            brand: "Kato",
            model: "NK-750V",
            carriageHeight: 3.0,
            pivotHeight: 2.8,
            baseSwingRadius: 2.1,
            baseBoomLen: 50.0,
            baseJibLen: 8.0,
            minBoomAngle: 3,
            maxBoomAngle: 80,
            minJibAngle: 0,
            maxJibAngle: 24,
            counterweightRadius: 4.9
        },
        manitowoc_mlc165: {
            id: "manitowoc_mlc165",
            name: "Manitowoc MLC165-1 (demo)",
            brand: "Manitowoc",
            model: "MLC165-1",
            carriageHeight: 0.0,   // crawler — boom foot near ground level
            pivotHeight: 25.7,     // lattice boom foot height above tracks
            baseSwingRadius: 3.2,
            baseBoomLen: 94.0,     // lattice main boom
            baseJibLen: 0.0,       // no jib in this demo pose
            minBoomAngle: 4,
            maxBoomAngle: 86,
            minJibAngle: 0,
            maxJibAngle: 0,
            counterweightRadius: 4.6
        }
    };
});