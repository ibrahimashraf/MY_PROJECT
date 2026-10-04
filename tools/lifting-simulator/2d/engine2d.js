// INTEGIN 2D Parametric Dynamic Blocks Engine
// Pure vector CAD canvas renderer with interactive kinematic handles
// Standard compliance: ASME B30.5, ISO 4309, BS 7121

class Vector2D {
    constructor(x = 0, y = 0) {
        this.x = x;
        this.y = y;
    }
    add(v) { return new Vector2D(this.x + v.x, this.y + v.y); }
    sub(v) { return new Vector2D(this.x - v.x, this.y - v.y); }
    mul(s) { return new Vector2D(this.x * s, this.y * s); }
    length() { return Math.hypot(this.x, this.y); }
    dist(v) { return Math.hypot(this.x - v.x, this.y - v.y); }
}

const LIFT_PHYSICS = {
    // Inverse kinematics: computes boom angle alpha from working radius
    radiusToBoomAngle(radiusM, boomLenM, pivotOffsetX = 0) {
        const dx = radiusM - pivotOffsetX;
        if (dx >= boomLenM) return 0.0;
        if (dx <= 0) return 90.0;
        const rad = Math.acos(dx / boomLenM);
        return (rad * 180.0) / Math.PI;
    },

    // Forward kinematics: computes working radius from boom angle
    boomAngleToRadius(boomAngleDeg, boomLenM, pivotOffsetX = 0) {
        const rad = (boomAngleDeg * Math.PI) / 180.0;
        return pivotOffsetX + boomLenM * Math.cos(rad);
    },

    // Computes hook height from boom angle and pivot height
    boomAngleToHookHeight(boomAngleDeg, boomLenM, pivotOffsetY = 0) {
        const rad = (boomAngleDeg * Math.PI) / 180.0;
        return pivotOffsetY + boomLenM * Math.sin(rad);
    },

    // Rigging: 2-leg sling tension with ASME B30.20 / BS 7121 dynamic factor
    calculateSlingTension(grossLoadT, slingAngleDeg, dynamicFactor = 1.15) {
        const angle = Math.max(0.1, Math.min(90.0, slingAngleDeg));
        const rad = (angle * Math.PI) / 180.0;
        const sinA = Math.sin(rad);
        const legTensionT = (grossLoadT / (2.0 * sinA)) * dynamicFactor;
        const criticalDerated = angle < 30.0;
        return {
            slingAngleDeg: angle,
            legTensionT: Math.round(legTensionT * 100) / 100,
            dynamicFactor,
            criticalDerated,
            status: criticalDerated ? 'CRITICAL_LOCKOUT' : (angle < 45.0 ? 'DERATED_WARNING' : 'SAFE')
        };
    },

    // Ground Bearing Pressure (GBP) via 4-point moment equilibrium
    calculateOutriggers(craneModel, craneState, grossLoadT, soilAllowableKP = 220) {
        const chassis = craneModel.chassis;
        const outriggers = craneModel.outriggers;
        const extFactor = outriggers.extensions[craneState.outriggerExt] || 1.0;
        
        const halfSpanX = (outriggers.spanXM * extFactor) / 2.0;
        const halfSpanZ = (outriggers.spanZM * extFactor) / 2.0;
        const matArea = craneState.matAreaM2 || outriggers.defaultMatAreaM2;

        const totalWeightT = chassis.weightT + craneState.counterweightT + grossLoadT;
        const radius = craneState.workingRadiusM;
        const slewRad = (craneState.slewAngleDeg * Math.PI) / 180.0;

        // Overturning moments
        const mx = grossLoadT * (radius * Math.cos(slewRad)) - (craneState.counterweightT * 3.5);
        const mz = grossLoadT * (radius * Math.sin(slewRad));

        const baseReaction = totalWeightT / 4.0;
        const deltaX = halfSpanX > 0 ? mx / (4.0 * halfSpanX) : 0;
        const deltaZ = halfSpanZ > 0 ? mz / (4.0 * halfSpanZ) : 0;

        // 4 corners: FL, FR, RR, RL
        const pFL = Math.max(0, baseReaction + deltaX - deltaZ);
        const pFR = Math.max(0, baseReaction + deltaX + deltaZ);
        const pRR = Math.max(0, baseReaction - deltaX + deltaZ);
        const pRL = Math.max(0, baseReaction - deltaX - deltaZ);

        const maxP = Math.max(pFL, pFR, pRR, pRL);
        const gbpKP = matArea > 0 ? (maxP * 9.80665) / matArea : 999.0;
        const fos = gbpKP > 0 ? soilAllowableKP / gbpKP : 0;

        return {
            reactionsT: {
                FL: Math.round(pFL * 10) / 10,
                FR: Math.round(pFR * 10) / 10,
                RR: Math.round(pRR * 10) / 10,
                RL: Math.round(pRL * 10) / 10
            },
            maxReactionT: Math.round(maxP * 10) / 10,
            matAreaM2: matArea,
            groundPressureKP: Math.round(gbpKP * 10) / 10,
            fos: Math.round(fos * 100) / 100,
            status: fos >= 1.5 ? 'PASS' : (fos >= 1.0 ? 'WARNING' : 'FAIL_OVERLOAD')
        };
    },

    // Step-down capacity lookup from OEM charts
    getRatedCapacity(craneModel, boomLenM, radiusM) {
        if (!craneModel || !craneModel.chart || craneModel.chart.length === 0) {
            return 0.0;
        }
        const cfg = craneModel.chart[0]; // Active configuration
        for (let i = 0; i < cfg.points.length; i++) {
            if (radiusM <= cfg.points[i].radiusM) {
                return cfg.points[i].capacityT;
            }
        }
        return 0.0; // Radius beyond chart boundary
    }
};

class CADViewport {
    constructor(canvas, isPlanView = true) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.isPlanView = isPlanView;
        this.scale = 14.0; // pixels per world meter
        this.pan = new Vector2D(canvas.width / 2, isPlanView ? canvas.height / 2 : canvas.height - 80);
        this.isDragging = false;
        this.dragGrip = null;
        this.grips = [];
    }

    worldToScreen(wx, wy) {
        if (this.isPlanView) {
            // Plan View: wx = World X (meters), wy = World Z (meters)
            return new Vector2D(this.pan.x + wx * this.scale, this.pan.y + wy * this.scale);
        } else {
            // Elevation View: wx = World X (meters), wy = World Y (height meters)
            return new Vector2D(this.pan.x + wx * this.scale, this.pan.y - wy * this.scale);
        }
    }

    screenToWorld(sx, sy) {
        if (this.isPlanView) {
            return new Vector2D((sx - this.pan.x) / this.scale, (sy - this.pan.y) / this.scale);
        } else {
            return new Vector2D((sx - this.pan.x) / this.scale, (this.pan.y - sy) / this.scale);
        }
    }

    clear() {
        const w = this.canvas.width;
        const h = this.canvas.height;
        this.ctx.fillStyle = '#0b0f19';
        this.ctx.fillRect(0, 0, w, h);
        this.drawGrid();
        this.grips = [];
    }

    drawGrid() {
        const ctx = this.ctx;
        const w = this.canvas.width;
        const h = this.canvas.height;

        // Minor grid (1m)
        ctx.strokeStyle = '#151e32';
        ctx.lineWidth = 1;
        const step1m = 1.0 * this.scale;
        const startX = this.pan.x % step1m;
        const startY = this.pan.y % step1m;

        ctx.beginPath();
        for (let x = startX; x < w; x += step1m) {
            ctx.moveTo(x, 0); ctx.lineTo(x, h);
        }
        for (let y = startY; y < h; y += step1m) {
            ctx.moveTo(0, y); ctx.lineTo(w, y);
        }
        ctx.stroke();

        // Major grid (5m)
        ctx.strokeStyle = '#1e2d4d';
        ctx.lineWidth = 1.2;
        const step5m = 5.0 * this.scale;
        const start5X = this.pan.x % step5m;
        const start5Y = this.pan.y % step5m;

        ctx.beginPath();
        for (let x = start5X; x < w; x += step5m) {
            ctx.moveTo(x, 0); ctx.lineTo(x, h);
        }
        for (let y = start5Y; y < h; y += step5m) {
            ctx.moveTo(0, y); ctx.lineTo(w, y);
        }
        ctx.stroke();

        // Axes
        ctx.strokeStyle = '#3b82f6';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        if (this.isPlanView) {
            ctx.moveTo(this.pan.x, 0); ctx.lineTo(this.pan.x, h); // Z axis
            ctx.moveTo(0, this.pan.y); ctx.lineTo(w, this.pan.y); // X axis
        } else {
            // Ground line
            ctx.strokeStyle = '#10b981';
            ctx.moveTo(0, this.pan.y); ctx.lineTo(w, this.pan.y);
            // Centerline
            ctx.strokeStyle = '#3b82f6';
            ctx.moveTo(this.pan.x, 0); ctx.lineTo(this.pan.x, h);
        }
        ctx.stroke();
    }

    registerGrip(id, worldPos, radiusPx, type) {
        const screenPos = this.worldToScreen(worldPos.x, worldPos.y);
        this.grips.push({ id, worldPos, screenPos, radiusPx, type });
        this.drawGrip(screenPos, type);
    }

    drawGrip(screenPos, type) {
        const ctx = this.ctx;
        ctx.save();
        ctx.fillStyle = type === 'active' ? '#f59e0b' : '#38bdf8';
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.arc(screenPos.x, screenPos.y, 6, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.restore();
    }

    findGrip(sx, sy) {
        for (const g of this.grips) {
            const d = Math.hypot(g.screenPos.x - sx, g.screenPos.y - sy);
            if (d <= g.radiusPx + 4) return g;
        }
        return null;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = { Vector2D, LIFT_PHYSICS, CADViewport };
}
