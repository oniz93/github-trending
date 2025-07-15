import 'package:flutter/material.dart';
import 'package:github_trending_app/models/repository.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'dart:ui';

class FullScreenRepoCard extends StatelessWidget {
  final Repository repository;

  const FullScreenRepoCard({super.key, required this.repository});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Stack(
        children: [
          Container(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                colors: [Colors.grey[900]!, Colors.black],
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildHeader(),
                const SizedBox(height: 16.0),
                _buildReadme(context),
                const Spacer(),
                _buildFooter(),
              ],
            ),
          ),
          _buildStats(context),
        ],
      ),
    );
  }

  Widget _buildHeader() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          repository.name,
          style: GoogleFonts.lora(
            fontSize: 32.0,
            fontWeight: FontWeight.bold,
            color: Colors.white,
          ),
        ),
        if (repository.description != null)
          Text(
            repository.description!,
            style: GoogleFonts.lora(
              fontSize: 16.0,
              color: Colors.white70,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
      ],
    );
  }

  Widget _buildReadme(BuildContext context) {
    return Expanded(
      child: ClipRRect(
        borderRadius: BorderRadius.circular(12.0),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 10.0, sigmaY: 10.0),
          child: Container(
            padding: const EdgeInsets.all(16.0),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.1),
              borderRadius: BorderRadius.circular(12.0),
              border: Border.all(color: Colors.white.withOpacity(0.2)),
            ),
            child: Markdown(
              data: repository.readmeSnippet ?? 'No README available.',
              styleSheet: MarkdownStyleSheet.fromTheme(Theme.of(context)).copyWith(
                p: GoogleFonts.openSans(color: Colors.white, fontSize: 14),
                h1: GoogleFonts.lora(color: Colors.white, fontSize: 24),
                h2: GoogleFonts.lora(color: Colors.white, fontSize: 20),
                code: GoogleFonts.robotoMono(backgroundColor: Colors.black.withOpacity(0.2), color: Colors.lightBlueAccent),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildFooter() {
    return Row(
      children: [
        CircleAvatar(
          backgroundImage: NetworkImage(repository.avatarUrl),
          radius: 20,
        ),
        const SizedBox(width: 12.0),
        Text(
          repository.owner.login,
          style: GoogleFonts.robotoMono(
            fontSize: 16.0,
            color: Colors.white,
          ),
        ),
      ],
    );
  }

  Widget _buildStats(BuildContext context) {
    return Positioned(
      right: 16.0,
      bottom: 100.0,
      child: Column(
        children: [
          _buildStatIcon(Icons.star, repository.stargazersCount.toString(), Colors.yellow),
          const SizedBox(height: 24.0),
          _buildStatIcon(Icons.call_split, repository.forksCount.toString(), Colors.blue),
          const SizedBox(height: 24.0),
          _buildStatIcon(Icons.share, 'Share', Colors.purple),
        ],
      ),
    );
  }

  Widget _buildStatIcon(IconData icon, String label, Color color) {
    return Column(
      children: [
        Icon(icon, color: color, size: 32.0),
        const SizedBox(height: 4.0),
        Text(
          label,
          style: GoogleFonts.robotoMono(
            fontSize: 14.0,
            color: Colors.white,
          ),
        ),
      ],
    );
  }
}
