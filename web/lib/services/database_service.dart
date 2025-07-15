import 'dart:io';
import 'package:flutter/services.dart';
import 'package:path/path.dart';
import 'package:sqflite/sqflite.dart';
import 'package:github_trending_app/models/repository.dart';

class DatabaseService {
  static final DatabaseService _instance = DatabaseService._internal();
  factory DatabaseService() => _instance;
  DatabaseService._internal();

  static Database? _database;

  Future<Database> get database async {
    if (_database != null) return _database!;
    _database = await _initDatabase();
    return _database!;
  }

  Future<Database> _initDatabase() async {
    final dbPath = await getDatabasesPath();
    final path = join(dbPath, 'app.db');

    await deleteDatabase(path);

    ByteData data = await rootBundle.load(join('assets', 'data', 'db.sqlite'));
    List<int> bytes = data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes);
    await File(path).writeAsBytes(bytes);

    return await openDatabase(path);
  }

  Future<List<Repository>> getRepositories({
    int limit = 20,
    int offset = 0,
    String? language,
  }) async {
    final db = await database;
    final List<Map<String, dynamic>> maps = await db.query(
      'repositories',
      where: language != null ? 'language = ?' : null,
      whereArgs: language != null ? [language] : null,
      orderBy: 'stargazers_count DESC',
      limit: limit,
      offset: offset,
    );
    return List.generate(maps.length, (i) {
      return Repository.fromMap(maps[i]);
    });
  }
}
