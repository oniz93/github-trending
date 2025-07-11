import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:github_trending_app/src/providers/repository_provider.dart';
import 'package:github_trending_app/src/views/repository_list_screen.dart';

void main() {
  runApp(
    ChangeNotifierProvider(
      create: (context) => RepositoryProvider(),
      child: const MyApp(),
    ),
  );
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
      home: const RepositoryListScreen(),
    );
  }
}