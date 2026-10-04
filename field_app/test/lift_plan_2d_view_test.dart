import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/presentation/lift_plan_2d_view.dart';

void main() {
  testWidgets('LiftPlan2DView renders plan view with badges and CAD canvas',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 800,
            height: 600,
            child: LiftPlan2DView(
              craneModel: 'Liebherr LTM 1500-8.1',
              boomLengthMeters: 36.9,
              workingRadiusMeters: 14.0,
              boomAngleDeg: 63.4,
              slewAngleDeg: 25.0,
              grossLoadTonnes: 45.0,
              ratedCapacityTonnes: 102.0,
              groundBearingFoS: 1.68,
              showPlanView: true,
            ),
          ),
        ),
      ),
    );

    // Verify model name and telemetry badges
    expect(find.text('Liebherr LTM 1500-8.1'), findsOneWidget);
    expect(find.text('R: 14.0m'), findsOneWidget);
    expect(find.text('α: 63.4°'), findsOneWidget);
    expect(find.text('Cap: 44%'), findsOneWidget);
    expect(find.text('FoS: 1.68'), findsOneWidget);

    // Verify CustomPaint is present
    expect(find.byType(CustomPaint), findsWidgets);
  });

  testWidgets('LiftPlan2DView renders elevation view correctly',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 800,
            height: 600,
            child: LiftPlan2DView(
              craneModel: 'Tadano ATF 400G-6',
              boomLengthMeters: 35.0,
              workingRadiusMeters: 16.0,
              boomAngleDeg: 60.0,
              slewAngleDeg: 0.0,
              grossLoadTonnes: 52.0,
              ratedCapacityTonnes: 52.0,
              groundBearingFoS: 1.15,
              showPlanView: false,
            ),
          ),
        ),
      ),
    );

    expect(find.text('Tadano ATF 400G-6'), findsOneWidget);
    expect(find.text('R: 16.0m'), findsOneWidget);
    expect(find.text('Cap: 100%'), findsOneWidget);
    expect(find.text('FoS: 1.15'), findsOneWidget);
  });
}
