import 'package:flutter/material.dart';
import 'package:github_trending_app/models/repository.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:transparent_image/transparent_image.dart';

class RepoCard extends StatelessWidget {
  final Repository repository;

  const RepoCard({super.key, required this.repository});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16.0),
      margin: const EdgeInsets.symmetric(vertical: 8.0, horizontal: 16.0),
      decoration: BoxDecoration(
        color: Colors.white.withOpacity(0.1),
        borderRadius: BorderRadius.circular(12.0),
        border: Border.all(color: Colors.white.withOpacity(0.2)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              CircleAvatar(
                backgroundImage: CachedNetworkImageProvider(repository.avatarUrl),
                radius: 20,
              ),
              const SizedBox(width: 12.0),
              Expanded(
                child: Text(
                  repository.fullName,
                  style: GoogleFonts.robotoMono(
                    fontSize: 18.0,
                    fontWeight: FontWeight.bold,
                    color: Colors.white,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12.0),
          if (repository.description != null)
            Text(
              repository.description!,
              style: GoogleFonts.openSans(
                fontSize: 14.0,
                color: Colors.white70,
              ),
            ),
          const SizedBox(height: 12.0),
          Row(
            children: [
              if (repository.language != null)
                Row(
                  children: [
                    Icon(Icons.code, color: Colors.white70, size: 16.0),
                    const SizedBox(width: 4.0),
                    Text(
                      repository.language!,
                      style: GoogleFonts.robotoMono(
                        fontSize: 12.0,
                        color: Colors.white70,
                      ),
                    ),
                  ],
                ),
              const Spacer(),
              Icon(Icons.star_border, color: Colors.white70, size: 16.0),
              const SizedBox(width: 4.0),
              Text(
                repository.stargazersCount.toString(),
                style: GoogleFonts.robotoMono(
                  fontSize: 12.0,
                  color: Colors.white70,
                ),
              ),
              const SizedBox(width: 16.0),
              Icon(Icons.call_split, color: Colors.white70, size: 16.0),
              const SizedBox(width: 4.0),
              Text(
                repository.forksCount.toString(),
                style: GoogleFonts.robotoMono(
                  fontSize: 12.0,
                  color: Colors.white70,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
