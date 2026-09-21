import 'package:drift/drift.dart';

class OutboxEntries extends Table {
  TextColumn get transactionId => text()();
  TextColumn get mutation => text()();
  TextColumn get state => text()();
  IntColumn get attempts => integer().withDefault(const Constant(0))();
  TextColumn get lastError => text().nullable()();
  TextColumn get chainHash => text().nullable()();
  TextColumn get acknowledgedAt => text().nullable()();
  TextColumn get createdAt => text()();

  @override
  Set<Column> get primaryKey => {transactionId};
}
