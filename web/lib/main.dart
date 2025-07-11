import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:github_trending_app/src/views/repository_list_view.dart';

void main() {
  runApp(const ProviderScope(child: MyApp()));
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'GitHub Trending',
      theme: ThemeData(
        primarySwatch: Colors.blue,
      ),
      home: const RepositoryListView(),
    );
  }
}