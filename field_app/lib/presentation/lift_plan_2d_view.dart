import 'dart:math' as math;
import 'package:flutter/material.dart';

/// LiftPlan2DView provides an offline 2D CAD vector rendering of crane lift plans
/// on rugged mobile field tablets (Item 3.13 / Hazard 12 air-gapped compliance).
class LiftPlan2DView extends StatelessWidget {
  final String craneModel;
  final double boomLengthMeters;
  final double workingRadiusMeters;
  final double boomAngleDeg;
  final double slewAngleDeg;
  final double grossLoadTonnes;
  final double ratedCapacityTonnes;
  final double groundBearingFoS;
  final bool showPlanView;

  const LiftPlan2DView({
    super.key,
    this.craneModel = 'Liebherr LTM 1500-8.1',
    this.boomLengthMeters = 36.9,
    this.workingRadiusMeters = 14.0,
    this.boomAngleDeg = 63.4,
    this.slewAngleDeg = 25.0,
    this.grossLoadTonnes = 45.0,
    this.ratedCapacityTonnes = 102.0,
    this.groundBearingFoS = 1.68,
    this.showPlanView = true,
  });

  @override
  Widget build(BuildContext context) {
    final utilPct = ratedCapacityTonnes > 0
        ? (grossLoadTonnes / ratedCapacityTonnes) * 100
        : 999.0;

    return Container(
      decoration: BoxDecoration(
        color: const Color(0xFF0B0F19),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: const Color(0xFF1E293B)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Telemetry Bar
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            decoration: const BoxDecoration(
              color: Color(0xFF0F172A),
              borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
              border: Border(bottom: BorderSide(color: Color(0xFF1E293B))),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(
                  craneModel,
                  style: const TextStyle(
                    color: Color(0xFF38BDF8),
                    fontWeight: FontWeight.bold,
                    fontSize: 12,
                  ),
                ),
                Row(
                  children: [
                    _buildBadge(
                      'R: ${workingRadiusMeters.toStringAsFixed(1)}m',
                      const Color(0xFF94A3B8),
                    ),
                    const SizedBox(width: 8),
                    _buildBadge(
                      'α: ${boomAngleDeg.toStringAsFixed(1)}°',
                      const Color(0xFF94A3B8),
                    ),
                    const SizedBox(width: 8),
                    _buildBadge(
                      'Cap: ${utilPct.toStringAsFixed(0)}%',
                      utilPct > 100
                          ? const Color(0xFFF87171)
                          : (utilPct > 85
                              ? const Color(0xFFFBBF24)
                              : const Color(0xFF34D399)),
                    ),
                    const SizedBox(width: 8),
                    _buildBadge(
                      'FoS: ${groundBearingFoS.toStringAsFixed(2)}',
                      groundBearingFoS < 1.0
                          ? const Color(0xFFF87171)
                          : (groundBearingFoS < 1.5
                              ? const Color(0xFFFBBF24)
                              : const Color(0xFF34D399)),
                    ),
                  ],
                ),
              ],
            ),
          ),
          // Vector CAD Canvas
          Expanded(
            child: ClipRect(
              child: CustomPaint(
                painter: _LiftPlanPainter(
                  boomLengthMeters: boomLengthMeters,
                  workingRadiusMeters: workingRadiusMeters,
                  boomAngleDeg: boomAngleDeg,
                  slewAngleDeg: slewAngleDeg,
                  isPlanView: showPlanView,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBadge(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withOpacity(0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withOpacity(0.4)),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: color,
          fontSize: 10,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _LiftPlanPainter extends CustomPainter {
  final double boomLengthMeters;
  final double workingRadiusMeters;
  final double boomAngleDeg;
  final double slewAngleDeg;
  final bool isPlanView;

  _LiftPlanPainter({
    required this.boomLengthMeters,
    required this.workingRadiusMeters,
    required this.boomAngleDeg,
    required this.slewAngleDeg,
    required this.isPlanView,
  });

  @override
  void paint(Canvas canvas, Size size) {
    const scale = 5.0; // pixels per meter
    final center = Offset(size.width / 2, size.height / 2);

    // Draw CAD Grid
    final gridPaint = Paint()
      ..color = const Color(0xFF151E32)
      ..strokeWidth = 1.0;

    const step = 5.0 * scale;
    for (double x = center.dx % step; x < size.width; x += step) {
      canvas.drawLine(Offset(x, 0), Offset(x, size.height), gridPaint);
    }
    for (double y = center.dy % step; y < size.height; y += step) {
      canvas.drawLine(Offset(0, y), Offset(size.width, y), gridPaint);
    }

    if (isPlanView) {
      _paintPlanView(canvas, center, scale);
    } else {
      _paintElevationView(canvas, Offset(size.width * 0.35, size.height - 40), scale);
    }
  }

  void _paintPlanView(Canvas canvas, Offset center, double scale) {
    // 1. Outriggers & Timber Mats
    final matPaint = Paint()..color = const Color(0xFFB45309);
    final outriggerPaint = Paint()
      ..color = const Color(0xFF475569)
      ..strokeWidth = 2.0;

    final spanX = 10.0 * scale;
    final spanZ = 9.6 * scale;
    final matSize = 3.0 * scale;

    final pads = [
      Offset(center.dx - spanX / 2, center.dy - spanZ / 2),
      Offset(center.dx + spanX / 2, center.dy - spanZ / 2),
      Offset(center.dx + spanX / 2, center.dy + spanZ / 2),
      Offset(center.dx - spanX / 2, center.dy + spanZ / 2),
    ];

    for (final pad in pads) {
      canvas.drawLine(center, pad, outriggerPaint);
      canvas.drawRect(
        Rect.fromCenter(center: pad, width: matSize, height: matSize),
        matPaint,
      );
    }

    // 2. Chassis
    final chassisPaint = Paint()..color = const Color(0xFF1E293B);
    final chassisBorder = Paint()
      ..color = const Color(0xFF64748B)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5;

    final chassisRect = Rect.fromCenter(
      center: center,
      width: 18.0 * scale,
      height: 3.0 * scale,
    );
    canvas.drawRect(chassisRect, chassisPaint);
    canvas.drawRect(chassisRect, chassisBorder);

    // 3. Working Radius Circle
    final radiusPaint = Paint()
      ..color = const Color(0xFF38BDF8)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.2;
    canvas.drawCircle(center, workingRadiusMeters * scale, radiusPaint);

    // 4. Slewed Boom Line
    final slewRad = (slewAngleDeg * math.pi) / 180.0;
    final hookOffset = Offset(
      center.dx + workingRadiusMeters * scale * math.cos(slewRad),
      center.dy + workingRadiusMeters * scale * math.sin(slewRad),
    );

    final boomPaint = Paint()
      ..color = const Color(0xFFF59E0B)
      ..strokeWidth = 3.5;
    canvas.drawLine(center, hookOffset, boomPaint);

    // Hook Point
    final hookPaint = Paint()..color = const Color(0xFF38BDF8);
    canvas.drawCircle(hookOffset, 5.0, hookPaint);
  }

  void _paintElevationView(Canvas canvas, Offset base, double scale) {
    // Ground line
    final groundPaint = Paint()
      ..color = const Color(0xFF10B981)
      ..strokeWidth = 2.0;
    canvas.drawLine(Offset(0, base.dy), Offset(base.dx * 3, base.dy), groundPaint);

    // Chassis
    final chassisPaint = Paint()..color = const Color(0xFF1E293B);
    final chassisRect = Rect.fromLTWH(
      base.dx - 9.0 * scale,
      base.dy - 2.5 * scale,
      18.0 * scale,
      2.5 * scale,
    );
    canvas.drawRect(chassisRect, chassisPaint);

    // Boom
    final rad = (boomAngleDeg * math.pi) / 180.0;
    final pivot = Offset(base.dx - 2.0 * scale, base.dy - 3.0 * scale);
    final tip = Offset(
      pivot.dx + boomLengthMeters * scale * math.cos(rad),
      pivot.dy - boomLengthMeters * scale * math.sin(rad),
    );

    final boomPaint = Paint()
      ..color = const Color(0xFFF59E0B)
      ..strokeWidth = 4.0;
    canvas.drawLine(pivot, tip, boomPaint);

    // Cable & Hook
    final hookY = base.dy - 6.0 * scale;
    final cablePaint = Paint()
      ..color = const Color(0xFF94A3B8)
      ..strokeWidth = 1.5;
    canvas.drawLine(tip, Offset(tip.dx, hookY), cablePaint);

    // Hook block
    final hookPaint = Paint()..color = const Color(0xFFFBBF24);
    canvas.drawRect(
      Rect.fromCenter(center: Offset(tip.dx, hookY), width: 8, height: 12),
      hookPaint,
    );
  }

  @override
  bool shouldRepaint(covariant _LiftPlanPainter oldDelegate) {
    return oldDelegate.workingRadiusMeters != workingRadiusMeters ||
        oldDelegate.boomAngleDeg != boomAngleDeg ||
        oldDelegate.slewAngleDeg != slewAngleDeg ||
        oldDelegate.isPlanView != isPlanView;
  }
}
