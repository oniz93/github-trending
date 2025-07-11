import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:github_trending_app/src/providers/repository_provider.dart';
import 'package:github_trending_app/src/widgets/repository_card.dart';
import 'package:scrollable_positioned_list/scrollable_positioned_list.dart';

class RepositoryListView extends ConsumerStatefulWidget {
  const RepositoryListView({super.key});

  @override
  _RepositoryListViewState createState() => _RepositoryListViewState();
}

class _RepositoryListViewState extends ConsumerState<RepositoryListView> {
  final ItemScrollController _scrollController = ItemScrollController();

  @override
  void initState() {
    super.initState();
    // Initial data fetch
    Future.microtask(() => ref.read(repositoryProvider.notifier).fetchRepositories());
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(repositoryProvider);

    return Scaffold(
      body: state.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stackTrace) => Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text('Error: $error'),
              ElevatedButton(
                onPressed: () => ref.read(repositoryProvider.notifier).fetchRepositories(),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (repositories) {
          return NotificationListener<ScrollNotification>(
            onNotification: (notification) {
              if (notification.metrics.pixels >=
                  notification.metrics.maxScrollExtent * 0.9) {
                ref.read(repositoryProvider.notifier).fetchRepositories();
              }
              return false;
            },
            child: ScrollablePositionedList.builder(
              itemCount: repositories.length,
              itemScrollController: _scrollController,
              itemBuilder: (context, index) {
                return RepositoryCard(repository: repositories[index]);
              },
            ),
          );
        },
      ),
    );
  }
}
