import 'package:flutter/material.dart';

enum ScreenType { compact, medium, expanded }

class AdaptiveScaffold extends StatelessWidget {
  const AdaptiveScaffold({
    super.key,
    required this.title,
    required this.body,
    this.secondaryBody,
    this.drawerItems,
    this.actions,
    this.selectedIndex = 0,
    this.onDestinationSelected,
  });

  final String title;
  final Widget body;
  final Widget? secondaryBody;
  final List<NavigationDestination>? drawerItems;
  final List<Widget>? actions;
  final int selectedIndex;
  final ValueChanged<int>? onDestinationSelected;

  static ScreenType getScreenType(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    if (width < 600) return ScreenType.compact;
    if (width < 1024) return ScreenType.medium;
    return ScreenType.expanded;
  }

  @override
  Widget build(BuildContext context) {
    final screenType = getScreenType(context);

    if (screenType == ScreenType.compact) {
      // Mobile Layout: Bottom navigation + Single Column
      return Scaffold(
        appBar: AppBar(
          title: Text(title),
          actions: actions,
        ),
        body: SafeArea(child: body),
        bottomNavigationBar: drawerItems != null && drawerItems!.isNotEmpty
            ? NavigationBar(
                selectedIndex: selectedIndex,
                onDestinationSelected: onDestinationSelected,
                destinations: drawerItems!,
              )
            : null,
      );
    }

    if (screenType == ScreenType.medium) {
      // Tablet Layout: Navigation Rail + Adaptive Main Body
      return Scaffold(
        appBar: AppBar(
          title: Text(title),
          actions: actions,
        ),
        body: Row(
          children: [
            if (drawerItems != null && drawerItems!.isNotEmpty)
              NavigationRail(
                selectedIndex: selectedIndex,
                onDestinationSelected: onDestinationSelected,
                labelType: NavigationRailLabelType.selected,
                destinations: drawerItems!
                    .map((d) => NavigationRailDestination(
                          icon: d.icon,
                          selectedIcon: d.selectedIcon,
                          label: Text(d.label),
                        ))
                    .toList(),
              ),
            const VerticalDivider(thickness: 1, width: 1),
            Expanded(child: SafeArea(child: body)),
          ],
        ),
      );
    }

    // Desktop/Web Layout: Master-Detail Split Pane + Permanent Sidebar
    return Scaffold(
      appBar: AppBar(
        title: Text('$title · Desktop Station'),
        actions: actions,
      ),
      body: Row(
        children: [
          if (drawerItems != null && drawerItems!.isNotEmpty)
            NavigationRail(
              selectedIndex: selectedIndex,
              onDestinationSelected: onDestinationSelected,
              extended: true,
              destinations: drawerItems!
                  .map((d) => NavigationRailDestination(
                        icon: d.icon,
                        selectedIcon: d.selectedIcon,
                        label: Text(d.label),
                      ))
                  .toList(),
            ),
          const VerticalDivider(thickness: 1, width: 1),
          // Master Pane
          Expanded(
            flex: secondaryBody != null ? 5 : 12,
            child: SafeArea(child: body),
          ),
          if (secondaryBody != null) ...[
            const VerticalDivider(thickness: 1, width: 1),
            // Detail / Secondary Pane
            Expanded(
              flex: 7,
              child: SafeArea(child: secondaryBody!),
            ),
          ],
        ],
      ),
    );
  }
}
