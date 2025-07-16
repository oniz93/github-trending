import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:github_trending_app/src/services/api_service.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class RepositoryCard extends ConsumerWidget {
  final dynamic repository;

  const RepositoryCard({super.key, required this.repository});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return GestureDetector(
      onHorizontalDragEnd: (details) {
        if (details.primaryVelocity! < 0) {
          // Swipe left
          ref.read(apiServiceProvider).trackOpenRepository(repository['id']);
          launchUrl(Uri.parse(repository['url']));
        }
      },
      child: Container(
        height: MediaQuery.of(context).size.height,
        padding: const EdgeInsets.all(16.0),
        child: Markdown(
          data: repository['readme'] ?? 'No README found.',
        ),
      ),
    );
  }
}
