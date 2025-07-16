// web/lib/src/views/repository_list_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:github_trending_app/src/providers/repository_provider.dart';
import 'package:github_trending_app/src/views/repository_card.dart';

class RepositoryListScreen extends StatefulWidget {
  const RepositoryListScreen({Key? key}) : super(key: key);

  @override
  State<RepositoryListScreen> createState() => _RepositoryListScreenState();
}

class _RepositoryListScreenState extends State<RepositoryListScreen> {
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    // Initial data fetch
    WidgetsBinding.instance.addPostFrameCallback((_) {
      Provider.of<RepositoryProvider>(context, listen: false).fetchRepositories(refresh: true);
    });

    _scrollController.addListener(() {
      if (_scrollController.position.pixels == _scrollController.position.maxScrollExtent) {
        // User has scrolled to the end, load more data
        Provider.of<RepositoryProvider>(context, listen: false).fetchRepositories();
      }
    });
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('GitHub Trending'),
      ),
      body: Consumer<RepositoryProvider>(
        builder: (context, provider, child) {
          if (provider.isLoading && provider.repositories.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (provider.errorMessage != null) {
            return Center(
              child: Text(
                provider.errorMessage!,
                style: const TextStyle(color: Colors.red),
              ),
            );
          }

          if (provider.repositories.isEmpty) {
            return const Center(child: Text('No repositories found.'));
          }

          return ListView.builder(
            controller: _scrollController,
            itemCount: provider.repositories.length + (provider.isLoading ? 1 : 0),
            itemBuilder: (context, index) {
              if (index == provider.repositories.length) {
                return const Padding(
                  padding: EdgeInsets.all(8.0),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              final repository = provider.repositories[index];
              return RepositoryCard(
                repository: repository,
                onCardTap: () {
                  // Track the open event
                  provider.trackOpenRepository(repository.id);
                  // Open the GitHub URL
                  // This is already handled by the TextButton inside RepositoryCard
                  // but if we wanted to navigate to an internal detail screen, we'd do it here.
                },
              );
            },
          );
        },
      ),
    );
  }
}
