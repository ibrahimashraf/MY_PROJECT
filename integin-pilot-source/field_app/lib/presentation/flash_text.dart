import 'package:flutter/material.dart';

/// A widget that shows [text] flashing rapidly (200 ms on/off).
///
/// The animation repeats indefinitely. Use it for brief attention‑grabbers
/// such as the introductory quote.
class FlashingText extends StatefulWidget {
  const FlashingText({
    super.key,
    required this.text,
    this.style,
  });

  final String text;
  final TextStyle? style;

  @override
  State<FlashingText> createState() => _FlashingTextState();
}

class _FlashingTextState extends State<FlashingText>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<double> _opacity;

  @override
  void initState() {
    super.initState();
    // Fast flash: 200 ms visible, 200 ms invisible → 400 ms period.
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 200),
    )..repeat(reverse: true);
    _opacity = Tween(begin: 1.0, end: 0.0).animate(_controller);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FadeTransition(
      opacity: _opacity,
      child: Text(
        widget.text,
        style: widget.style ??
            Theme.of(context)
                .textTheme
                .headlineSmall
                ?.copyWith(color: Colors.redAccent),
        textAlign: TextAlign.center,
      ),
    );
  }
}
