// web/lib/src/views/repository_card.dart
import 'package:flutter/material.dart';
import 'package:github_trending_app/src/models/repository.dart';
import 'package:url_launcher/url_launcher.dart';

class RepositoryCard extends StatelessWidget {
  final Repository repository;
  final VoidCallback onCardTap;

  const RepositoryCard({
    Key? key,
    required this.repository,
    required this.onCardTap,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(8.0),
      child: InkWell(
        onTap: onCardTap,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  CircleAvatar(
                    backgroundImage: NetworkImage(repository.ownerAvatarUrl),
                    radius: 20,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      repository.fullName,
                      style: const TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                      ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              if (repository.description != null)
                Text(
                  repository.description!,
                  style: const TextStyle(fontSize: 14),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              const SizedBox(height: 10),
              Row(
                children: [
                  const Icon(Icons.star, size: 16, color: Colors.amber),
                  const SizedBox(width: 5),
                  Text('${repository.stargazersCount}'),
                  const SizedBox(width: 20),
                  if (repository.language != null)
                    Row(
                      children: [
                        const Icon(Icons.code, size: 16),
                        const SizedBox(width: 5),
                        Text(repository.language!),
                      ],
                    ),
                ],
              ),
              const SizedBox(height: 10),
              Align(
                alignment: Alignment.bottomRight,
                child: TextButton(
                  onPressed: () async {
                    if (await canLaunchUrl(Uri.parse(repository.htmlUrl))) {
                      await launchUrl(Uri.parse(repository.htmlUrl));
                    }
                  },
                  child: const Text('View on GitHub'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
