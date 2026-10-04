// INTEGIN Parametric Dynamic Blocks Catalog
// Governing specification: docs/architecture/INTEGIN_LIFTING_PLAN_SIMULATOR_2D_3D_4D_SPECIFICATION.md
// Covers major crane OEMs: Liebherr, Tadano, Demag, Kato, Manitowoc.

const CRANE_BLOCKS_CATALOG = {
    'LIEBHERR_LTM_1500': {
        id: 'LIEBHERR_LTM_1500',
        manufacturer: 'Liebherr',
        model: 'LTM 1500-8.1',
        craneClass: 'All-Terrain Mobile Crane',
        chassis: { lengthM: 21.4, widthM: 3.0, weightT: 96.0 },
        outriggers: {
            spanXM: 10.0,
            spanZM: 9.6,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 4.0
        },
        boom: {
            minLenM: 16.1,
            maxLenM: 84.0,
            pivotOffset: { x: -2.5, y: 3.2, z: 0.0 },
            sections: [16.1, 26.5, 36.9, 47.3, 57.7, 68.1, 78.5, 84.0]
        },
        counterweights: [15, 45, 90, 135, 165],
        chart: [
            {
                boomLen: 36.9,
                counterweightT: 90,
                points: [
                    { radiusM: 7.0, capacityT: 185.0 },
                    { radiusM: 10.0, capacityT: 145.0 },
                    { radiusM: 14.0, capacityT: 102.0 },
                    { radiusM: 18.0, capacityT: 76.0 },
                    { radiusM: 22.0, capacityT: 58.0 },
                    { radiusM: 26.0, capacityT: 46.0 },
                    { radiusM: 30.0, capacityT: 38.0 },
                    { radiusM: 34.0, capacityT: 31.0 }
                ]
            }
        ]
    },
    'LIEBHERR_LTM_1100': {
        id: 'LIEBHERR_LTM_1100',
        manufacturer: 'Liebherr',
        model: 'LTM 1100-4.2',
        craneClass: 'All-Terrain Mobile Crane',
        chassis: { lengthM: 13.98, widthM: 2.75, weightT: 48.0 },
        outriggers: {
            spanXM: 8.58,
            spanZM: 7.0,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 2.5
        },
        boom: {
            minLenM: 11.5,
            maxLenM: 60.0,
            pivotOffset: { x: -1.8, y: 2.8, z: 0.0 },
            sections: [11.5, 22.6, 33.8, 45.0, 52.5, 60.0]
        },
        counterweights: [4.4, 8.8, 17.2, 28.2],
        chart: [
            {
                boomLen: 33.8,
                counterweightT: 28.2,
                points: [
                    { radiusM: 4.0, capacityT: 75.0 },
                    { radiusM: 6.0, capacityT: 55.0 },
                    { radiusM: 8.0, capacityT: 42.0 },
                    { radiusM: 12.0, capacityT: 26.5 },
                    { radiusM: 16.0, capacityT: 18.2 },
                    { radiusM: 20.0, capacityT: 13.4 },
                    { radiusM: 24.0, capacityT: 10.1 },
                    { radiusM: 28.0, capacityT: 7.8 }
                ]
            }
        ]
    },
    'TADANO_ATF_400G': {
        id: 'TADANO_ATF_400G',
        manufacturer: 'Tadano',
        model: 'ATF 400G-6',
        craneClass: 'All-Terrain Mobile Crane',
        chassis: { lengthM: 17.9, widthM: 3.0, weightT: 72.0 },
        outriggers: {
            spanXM: 8.9,
            spanZM: 8.5,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 3.5
        },
        boom: {
            minLenM: 15.0,
            maxLenM: 60.0,
            pivotOffset: { x: -2.0, y: 3.0, z: 0.0 },
            sections: [15.0, 25.0, 35.0, 45.0, 55.0, 60.0]
        },
        counterweights: [10, 25, 55, 95, 138],
        chart: [
            {
                boomLen: 35.0,
                counterweightT: 95,
                points: [
                    { radiusM: 5.0, capacityT: 140.0 },
                    { radiusM: 8.0, capacityT: 105.0 },
                    { radiusM: 12.0, capacityT: 72.0 },
                    { radiusM: 16.0, capacityT: 52.0 },
                    { radiusM: 20.0, capacityT: 39.5 },
                    { radiusM: 24.0, capacityT: 31.0 },
                    { radiusM: 28.0, capacityT: 24.8 },
                    { radiusM: 32.0, capacityT: 20.1 }
                ]
            }
        ]
    },
    'TADANO_ATF_200G_5': {
        id: 'TADANO_ATF_200G_5',
        manufacturer: 'Tadano',
        model: 'ATF 200G-5',
        craneClass: 'All-Terrain Mobile Crane',
        chassis: { lengthM: 15.2, widthM: 3.0, weightT: 60.0 },
        outriggers: {
            spanXM: 8.5,
            spanZM: 8.3,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 3.0
        },
        boom: {
            minLenM: 13.2,
            maxLenM: 68.0,
            pivotOffset: { x: -1.9, y: 2.9, z: 0.0 },
            sections: [13.2, 22.0, 30.6, 40.0, 50.0, 60.0, 68.0]
        },
        counterweights: [12, 25, 38, 50],
        chart: [
            {
                boomLen: 30.6,
                counterweightT: 50,
                points: [
                    { radiusM: 4.0, capacityT: 110.0 },
                    { radiusM: 7.0, capacityT: 75.0 },
                    { radiusM: 10.0, capacityT: 53.0 },
                    { radiusM: 14.0, capacityT: 35.0 },
                    { radiusM: 18.0, capacityT: 25.0 },
                    { radiusM: 22.0, capacityT: 18.5 },
                    { radiusM: 26.0, capacityT: 14.0 }
                ]
            }
        ]
    },
    'DEMAG_AC_200_1': {
        id: 'DEMAG_AC_200_1',
        manufacturer: 'Demag',
        model: 'AC 200-1',
        craneClass: 'All-Terrain Mobile Crane',
        chassis: { lengthM: 14.8, widthM: 3.0, weightT: 60.0 },
        outriggers: {
            spanXM: 8.5,
            spanZM: 8.2,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 3.0
        },
        boom: {
            minLenM: 12.4,
            maxLenM: 67.8,
            pivotOffset: { x: -1.9, y: 2.9, z: 0.0 },
            sections: [12.4, 21.0, 29.2, 38.0, 48.0, 58.0, 67.8]
        },
        counterweights: [15, 30, 45, 69],
        chart: [
            {
                boomLen: 29.2,
                counterweightT: 69,
                points: [
                    { radiusM: 4.0, capacityT: 115.0 },
                    { radiusM: 7.0, capacityT: 78.0 },
                    { radiusM: 10.0, capacityT: 56.0 },
                    { radiusM: 14.0, capacityT: 37.0 },
                    { radiusM: 18.0, capacityT: 26.5 },
                    { radiusM: 22.0, capacityT: 19.8 },
                    { radiusM: 26.0, capacityT: 15.2 }
                ]
            }
        ]
    },
    'KATO_CR_250': {
        id: 'KATO_CR_250',
        manufacturer: 'Kato',
        model: 'CR-250',
        craneClass: 'City Rough Terrain Crane',
        chassis: { lengthM: 9.1, widthM: 2.4, weightT: 24.0 },
        outriggers: {
            spanXM: 6.3,
            spanZM: 6.0,
            extensions: { '100%': 1.0, '75%': 0.75, '50%': 0.5, '0%': 0.0 },
            defaultMatAreaM2: 1.5
        },
        boom: {
            minLenM: 6.7,
            maxLenM: 28.0,
            pivotOffset: { x: -1.2, y: 2.2, z: 0.0 },
            sections: [6.7, 12.0, 18.0, 23.5, 28.0]
        },
        counterweights: [3.5],
        chart: [
            {
                boomLen: 18.0,
                counterweightT: 3.5,
                points: [
                    { radiusM: 3.0, capacityT: 25.0 },
                    { radiusM: 5.0, capacityT: 18.0 },
                    { radiusM: 8.0, capacityT: 10.5 },
                    { radiusM: 12.0, capacityT: 5.8 },
                    { radiusM: 16.0, capacityT: 3.2 }
                ]
            }
        ]
    },
    'MANITOWOC_MLC300': {
        id: 'MANITOWOC_MLC300',
        manufacturer: 'Manitowoc',
        model: 'MLC300',
        craneClass: 'Crawler Crane with VPC',
        chassis: { lengthM: 12.5, widthM: 8.2, weightT: 145.0 },
        outriggers: {
            spanXM: 8.2,
            spanZM: 10.5,
            extensions: { '100%': 1.0, '75%': 1.0, '50%': 1.0, '0%': 1.0 },
            defaultMatAreaM2: 6.0
        },
        boom: {
            minLenM: 30.0,
            maxLenM: 96.0,
            pivotOffset: { x: -1.5, y: 2.5, z: 0.0 },
            sections: [30.0, 42.0, 54.0, 66.0, 78.0, 90.0, 96.0]
        },
        counterweights: [50, 100, 150, 200],
        chart: [
            {
                boomLen: 42.0,
                counterweightT: 150,
                points: [
                    { radiusM: 8.0, capacityT: 260.0 },
                    { radiusM: 12.0, capacityT: 195.0 },
                    { radiusM: 16.0, capacityT: 140.0 },
                    { radiusM: 20.0, capacityT: 108.0 },
                    { radiusM: 26.0, capacityT: 78.0 },
                    { radiusM: 32.0, capacityT: 59.0 },
                    { radiusM: 38.0, capacityT: 46.0 }
                ]
            }
        ]
    }
};

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { CRANE_BLOCKS_CATALOG };
}
