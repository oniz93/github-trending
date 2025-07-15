import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:github_trending_app/providers/repository_provider.dart';
import 'package:github_trending_app/widgets/full_screen_repo_card.dart';

class RepositoryListScreen extends StatefulWidget {
  const RepositoryListScreen({super.key});

  @override
  State<RepositoryListScreen> createState() => _RepositoryListScreenState();
}

class _RepositoryListScreenState extends State<RepositoryListScreen> {
  final PageController _pageController = PageController();

  @override
  void initState() {
    super.initState();
    final repositoryProvider = Provider.of<RepositoryProvider>(context, listen: false);
    repositoryProvider.fetchRepositories();
  }

  @override
  Widget build(BuildContext context) {
    final repositoryProvider = Provider.of<RepositoryProvider>(context);

    return Scaffold(
      body: repositoryProvider.repositories.isEmpty && repositoryProvider.isLoading
          ? const Center(child: CircularProgressIndicator())
          : PageView.builder(
              controller: _pageController,
              scrollDirection: Axis.vertical,
              itemCount: repositoryProvider.repositories.length,
              itemBuilder: (context, index) {
                final repository = repositoryProvider.repositories[index];
                return FullScreenRepoCard(repository: repository);
              },
              onPageChanged: (index) {
                if (index >= repositoryProvider.repositories.length - 5) {
                  repositoryProvider.fetchRepositories();
                }
                final repository = repositoryProvider.repositories[index];
                repositoryProvider.trackOpenRepository(int.parse(repository.id));
                if (repository.readmeSnippet == null) {
                  repositoryProvider.fetchReadmeForRepository(repository);
                }
              },
            ),
    );
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }
}