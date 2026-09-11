/* app.js — 2D Parametric Dynamic Blocks Lifting Simulator (planning aid only).
 *
 * Static, browser-native, zero backend, zero dependencies.
 *
 * IMPORTANT — physics honesty (AGENTS.md invariant 5):
 *   This tool draws geometry and solves simple radius/height trigonometry so a
 *   user can position and visualise the crane envelope in plan and elevation.
 *   It performs NO load-rating math of any kind. Nothing here is lift-rated.
 *   Certified capacity / envelope points (CENPs) may only be computed by the
 *   CEL nanocell. All output is labellable as a PLANNING AID ONLY.
 */

/* ------------------------------------------------------------------ */
/* Pure logic — no DOM access. Kept testable by the shim harness.      */
/* ------------------------------------------------------------------ */

var D2R = Math.PI / 180;

function getCrane(modelId) {
    var c = (typeof CRANES !== "undefined" && CRANES) ? CRANES[modelId] : null;
    if (!c) return null;
    var nums = [c.carriageHeight, c.pivotHeight, c.baseSwingRadius,
        c.baseBoomLen, c.baseJibLen, c.minBoomAngle, c.maxBoomAngle,
        c.minJibAngle, c.maxJibAngle, c.counterweightRadius];
    for (var i = 0; i < nums.length; i++) {
        if (typeof nums[i] !== "number" || !isFinite(nums[i])) return null;
    }
    if (c.baseBoomLen <= 0 || c.maxBoomAngle <= c.minBoomAngle) return null;
    return c;
}

function clamp(v, lo, hi) {
    if (v < lo) return lo;
    if (v > hi) return hi;
    return v;
}

function createState(modelId) {
    var crane = getCrane(modelId);
    if (!crane) {
        return {
            valid: false,
            error: "Invalid / unavailable crane model: \"" + String(modelId) + "\". Refusing to render. Check cranes.js."
        };
    }
    return {
        valid: true,
        modelId: crane.id,
        crane: crane,
        slewDeg: 45,
        boomDeg: Math.round(0.5 * (crane.minBoomAngle + crane.maxBoomAngle)),
        jibDeg: clamp(crane.minJibAngle, 0, crane.maxJibAngle)
    };
}

function setBoomAngle(st, deg) {
    if (!st.valid) return st;
    st.boomDeg = clamp(deg, st.crane.minBoomAngle, st.crane.maxBoomAngle);
    return st;
}

function setJibAngle(st, deg) {
    if (!st.valid) return st;
    st.jibDeg = clamp(deg, st.crane.minJibAngle, st.crane.maxJibAngle);
    return st;
}

function setSlewAngle(st, deg) {
    if (!st.valid) return st;
    st.slewDeg = ((deg % 360) + 360) % 360;
    return st;
}

/* Horizontal reach (load radius) and tip height from pure geometry.
 * Returns null for an invalid state. No load rating is produced here. */
function computeElevation(st) {
    if (!st || !st.valid) return null;
    var c = st.crane;
    var boomHyp = c.baseBoomLen * Math.cos(st.boomDeg * D2R);
    var boomV = c.baseBoomLen * Math.sin(st.boomDeg * D2R);
    var rel = (st.boomDeg + st.jibDeg) * D2R;
    var jibH = c.baseJibLen * Math.cos(rel);
    var jibV = c.baseJibLen * Math.sin(rel);
    return {
        radius: c.baseSwingRadius + boomHyp + jibH,
        tipHeight: c.carriageHeight + c.pivotHeight + boomV + jibV
    };
}

/* Radial distance between a point (px,py) and the slewing line drawn at
 * angle deg (0 deg = +X axis) through the turret centre. Used for plan-view
 * drag hit-testing. */
function distanceToRay(px, py, deg) {
    return Math.abs(Math.sin(deg * D2R) * px - Math.cos(deg * D2R) * py);
}

function nearestDragHandle(x, y, angle, maxPx) {
    var d = distanceToRay(x - 0, y - 0, angle);
    return d <= maxPx;
}

/* ------------------------------------------------------------------ */
/* DOM wiring — only runs inside a real browser.                       */
/* ------------------------------------------------------------------ */

if (typeof document !== "undefined") {
    (function () {
        var elSelect = document.getElementById("crane-model");
        var elStatus = document.getElementById("status");
        var elDisclaimer = document.getElementById("disclaimer");
        var elReadout = document.getElementById("readout");
        var canvasPlan = document.getElementById("canvas-plan");
        var canvasElv = document.getElementById("canvas-elevation");
        var views = [canvasPlan, canvasElv];

        var state = null;
        var dragging = null; // { view: "plan"|"elevation", kind: "slew"|"boom"|"jib" }

        function failClosed(msg) {
            /* Fail-closed: show plain text, disable views, never crash. */
            if (elStatus) elStatus.textContent = msg;
            if (views) views.forEach(function (cv) { cv.classList.add("disabled"); });
            if (elReadout) elReadout.textContent = "Spot position unavailable.";
            if (elSelect && elSelect.value && state && state.valid) {
                /* Keep selector usable to recover, but the state is void. */
            }
            state = { valid: false, error: msg };
            drawAll();
        }

        function selectModel(modelId) {
            var st = createState(modelId);
            if (!st.valid) {
                failClosed(st.error);
                return;
            }
            state = st;
            if (elStatus) elStatus.textContent = "";
            if (views) views.forEach(function (cv) { cv.classList.remove("disabled"); });
            drawAll();
        }

        function updateReadout() {
            if (!state || !state.valid || !elReadout) return;
            var g = computeElevation(state);
            if (!g) return;
            elReadout.textContent =
                state.crane.name + " | boom " + state.boomDeg.toFixed(1) +
                " deg | jib off " + state.jibDeg.toFixed(1) +
                " deg | slew " + state.slewDeg.toFixed(0) +
                " deg | load radius " + g.radius.toFixed(1) + " m | tip height " +
                g.tipHeight.toFixed(1) + " m";
        }

        function setupSelect() {
            if (!elSelect) return;
            elSelect.addEventListener("change", function () {
                selectModel(elSelect.value);
            });
            selectModel(elSelect.value);
        }

        /* --- canvas helpers -------------------------------------------------- */

        function getCtx(cv) {
            return cv.getContext("2d");
        }

        function clear(cv, color) {
            var ctx = getCtx(cv);
            ctx.setTransform(1, 0, 0, 1, 0, 0);
            ctx.fillStyle = color || "#0f141a";
            ctx.fillRect(0, 0, cv.width, cv.height);
        }

        function style(cv, color, width) {
            var ctx = getCtx(cv);
            ctx.strokeStyle = color;
            ctx.lineWidth = width;
            ctx.lineCap = "round";
        }

        function angleToPoint(cx, cy, len, deg) {
            return { x: cx + len * Math.cos(deg * D2R), y: cy - len * Math.sin(deg * D2R) };
        }

        function drawHandle(cv, cx, cy, deg, len, label) {
            var ctx = getCtx(cv);
            var p = angleToPoint(cx, cy, len, deg);
            ctx.fillStyle = "#41ff9b";
            ctx.beginPath();
            ctx.arc(p.x, p.y, 7, 0, Math.PI * 2);
            ctx.fill();
            ctx.strokeStyle = "#0f141a";
            ctx.lineWidth = 2;
            ctx.stroke();
            ctx.fillStyle = "#e8eef5";
            ctx.font = "11px system-ui, sans-serif";
            ctx.fillText(label, p.x - 4, p.y - 12);
        }

        /* --- elevation view -------------------------------------------------- */

        function drawElevation() {
            var cv = canvasElv;
            clear(cv);
            var W = cv.width, H = cv.height;
            var ctx = getCtx(cv);
            var groundY = H - 40;
            var pxPerM = Math.min(W / 130, H / 140);

            /* ground line */
            style(cv, "#2a3642", 2);
            ctx.beginPath();
            ctx.moveTo(0, groundY);
            ctx.lineTo(W, groundY);
            ctx.stroke();

            if (!state || !state.valid) {
                ctx.fillStyle = "#ff7a70";
                ctx.font = "600 14px system-ui, sans-serif";
                ctx.fillText("No crane model loaded.", 20, 40);
                return;
            }

            var c = state.crane;
            var groundBox = state.valid;
            var rotX = W * 0.5;
            var deckY = groundY - c.carriageHeight * pxPerM;
            var pivotX = rotX + c.baseSwingRadius * pxPerM;
            var pivotY = deckY - c.pivotHeight * pxPerM;

            /* carriage */
            if (groundBox) {
                ctx.fillStyle = "#33414e";
                ctx.fillRect(rotX - c.counterweightRadius * pxPerM - 8, deckY, (c.counterweightRadius * 2 + 16), 8);
            }

            /* counterweight block */
            ctx.fillStyle = "#ffb02e";
            ctx.fillRect(rotX - c.counterweightRadius * pxPerM - 6, deckY - 18, 12, 18);

            /* boom */
            var boomTip = angleToPoint(pivotX, pivotY, c.baseBoomLen * pxPerM, state.boomDeg);
            var jibAngle = state.boomDeg + state.jibDeg;
            var jibEnd = angleToPoint(boomTip.x, boomTip.y, c.baseJibLen * pxPerM, jibAngle);

            style(cv, "#f0c04a", 6);
            ctx.beginPath();
            ctx.moveTo(pivotX, pivotY);
            ctx.lineTo(boomTip.x, boomTip.y);
            ctx.stroke();

            style(cv, "#ff8c42", 8);
            ctx.beginPath();
            ctx.moveTo(boomTip.x, boomTip.y);
            ctx.lineTo(jibEnd.x, jibEnd.y);
            ctx.stroke();

            /* hook line */
            style(cv, "#9fb4c7", 1);
            ctx.beginPath();
            ctx.moveTo(jibEnd.x, jibEnd.y);
            ctx.lineTo(jibEnd.x, Math.min(groundY, jibEnd.y + 28 * pxPerM));
            ctx.stroke();

            /* load radius marker on ground */
            var g = computeElevation(state);
            if (g) {
                style(cv, "#41ff9b", 2);
                ctx.beginPath();
                ctx.moveTo(rotX, groundY);
                ctx.lineTo(rotX + g.radius * pxPerM, groundY);
                ctx.stroke();
                ctx.fillStyle = "#41ff9b";
                ctx.font = "600 12px system-ui, sans-serif";
                ctx.fillText(g.radius.toFixed(1) + " m", rotX + g.radius * pxPerM * 0.4, groundY - 8);
            }

            drawHandle(cv, pivotX, pivotY, state.boomDeg, (c.baseBoomLen - 6) * pxPerM, "B");
            if (c.baseJibLen > 0) {
                drawHandle(cv, boomTip.x, boomTip.y, jibAngle, (c.baseJibLen - 5) * pxPerM, "J");
            } else {
                ctx.fillStyle = "#e8eef5";
                ctx.font = "11px system-ui, sans-serif";
                ctx.fillText("no jib in this demo pose", boomTip.x - 90, boomTip.y - 14);
            }
        }

        /* --- plan view ------------------------------------------------------- */

        function drawPlan() {
            var cv = canvasPlan;
            clear(cv);
            var W = cv.width, H = cv.height;
            var ctx = getCtx(cv);
            var cx = W * 0.5, cy = H * 0.5;

            if (!state || !state.valid) {
                ctx.fillStyle = "#ff7a70";
                ctx.font = "600 14px system-ui, sans-serif";
                ctx.fillText("No crane model loaded.", 20, 40);
                return;
            }

            var c = state.crane;
            var pxPerM = Math.min(W / 140, H / 140);

            /* slew circle */
            style(cv, "#2a3642", 1);
            ctx.beginPath();
            ctx.arc(cx, cy, c.counterweightRadius * pxPerM, 0, Math.PI * 2);
            ctx.stroke();

            /* current boom ray = horizontal projection in plan */
            var proj = c.baseBoomLen * Math.cos(state.boomDeg * D2R) +
                c.baseJibLen * Math.cos((state.boomDeg + state.jibDeg) * D2R);
            var tip = angleToPoint(cx, cy, proj * pxPerM, state.slewDeg);

            style(cv, "#41d9ff", 2);
            ctx.beginPath();
            ctx.moveTo(cx, cy);
            ctx.lineTo(tip.x, tip.y);
            ctx.stroke();

            /* turret dot */
            ctx.fillStyle = "#ffb02e";
            ctx.beginPath();
            ctx.arc(cx, cy, 6, 0, Math.PI * 2);
            ctx.fill();

            drawHandle(cv, cx, cy, state.slewDeg, (proj - 8) * pxPerM, "S");

            /* radius arc */
            var g = computeElevation(state);
            if (g) {
                style(cv, "#41ff9b", 1);
                ctx.beginPath();
                ctx.arc(cx, cy, g.radius * pxPerM, 0, Math.PI * 2);
                ctx.stroke();
                ctx.fillStyle = "#41ff9b";
                ctx.font = "600 12px system-ui, sans-serif";
                ctx.fillText("slew " + state.slewDeg.toFixed(0) + " deg", tip.x + 10, tip.y + 4);
            }
        }

        function drawAll() {
            drawPlan();
            drawElevation();
            updateReadout();
        }

        /* --- pointer interaction ---------------------------------------------- */

        function handlePointerDown(view, ev) {
            if (!state || !state.valid) return;
            var cv = view === "plan" ? canvasPlan : canvasElv;
            var rect = cv.getBoundingClientRect();
            var x = ev.clientX - rect.left, y = ev.clientY - rect.top;

            if (view === "plan") {
                if (nearestDragHandle(x - cv.width / 2, y - cv.height / 2, state.slewDeg, 14)) {
                    dragging = { view: "plan", kind: "slew" };
                }
            } else {
                var c = state.crane;
                var groundY = cv.height - 40;
                var pxPerM = Math.min(cv.width / 130, cv.height / 140);
                var pivotX = cv.width * 0.5 + c.baseSwingRadius * pxPerM;
                var pivotY = groundY - c.carriageHeight * pxPerM - c.pivotHeight * pxPerM;
                var boomTip = angleToPoint(pivotX, pivotY, c.baseBoomLen * pxPerM, state.boomDeg);
                var jibAngle = state.boomDeg + state.jibDeg;
                var jibTip = angleToPoint(boomTip.x, boomTip.y, c.baseJibLen * pxPerM, jibAngle);

                var dBoom = Math.hypot(x - boomTip.x, y - boomTip.y);
                var dJib = Math.hypot(x - jibTip.x, y - jibTip.y);
                if (c.baseJibLen > 0 && dJib <= 14) {
                    dragging = { view: "elevation", kind: "jib" };
                } else if (dBoom <= 14) {
                    dragging = { view: "elevation", kind: "boom" };
                }
            }
            if (dragging) {
                view === "plan" ? canvasPlan : canvasElv;
                var tgt = view === "plan" ? canvasPlan : canvasElv;
                tgt.setPointerCapture(ev.pointerId);
            }
        }

        function handlePointerMove(view, ev) {
            if (!state || !state.valid || !dragging) return;
            var cv = view === "plan" ? canvasPlan : canvasElv;
            var rect = cv.getBoundingClientRect();
            var x = ev.clientX - rect.left, y = ev.clientY - rect.top;

            if (view === "plan" && dragging.kind === "slew") {
                var deg = Math.atan2(cv.height / 2 - y, x - cv.width / 2) / D2R;
                setSlewAngle(state, deg);
            } else if (view === "elevation") {
                var c = state.crane;
                var groundY = cv.height - 40;
                var pxPerM = Math.min(cv.width / 130, cv.height / 140);
                var pivotX = cv.width * 0.5 + c.baseSwingRadius * pxPerM;
                var pivotY = groundY - c.carriageHeight * pxPerM - c.pivotHeight * pxPerM;
                var ang = Math.atan2(pivotY - y, x - pivotX) / D2R;

                if (dragging.kind === "boom") setBoomAngle(state, ang);
                else if (dragging.kind === "jib") {
                    var bt = angleToPoint(pivotX, pivotY, c.baseBoomLen * pxPerM, state.boomDeg);
                    setJibAngle(state, Math.atan2(bt.y - y, x - bt.x) / D2R - state.boomDeg);
                }
            }
            drawAll();
        }

        function handlePointerEnd(ev) {
            if (dragging) {
                var cv = dragging.view === "plan" ? canvasPlan : canvasElv;
                try { cv.releasePointerCapture(ev.pointerId); } catch (e) { /* noop */ }
                dragging = null;
            }
        }

        function bindCanvas(cv, view) {
            cv.addEventListener("pointerdown", function (e) { handlePointerDown(view, e); });
            cv.addEventListener("pointermove", function (e) { handlePointerMove(view, e); });
            cv.addEventListener("pointerup", function (e) { handlePointerEnd(e); });
            cv.addEventListener("pointercancel", function (e) { handlePointerEnd(e); });
        }

        /* --- boot ------------------------------------------------------------- */

        function init() {
            if (elDisclaimer) {
                elDisclaimer.textContent =
                    "PLANNING AID ONLY — not lift-rated. Geometry/radius display only. " +
                    "No load calculation performed here. Certified capacity/envelope points " +
                    "(CENPs) require the CEL nanocell process.";
            }
            bindCanvas(canvasPlan, "plan");
            bindCanvas(canvasElv, "elevation");
            setupSelect();
        }

        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", init);
        } else {
            init();
        }
    })();
}