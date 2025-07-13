// web/lib/src/models/repository.dart
class Repository {
  final int id;
  final String name;
  final String fullName;
  final String ownerLogin;
  final String ownerAvatarUrl;
  final String htmlUrl;
  final String? description;
  final int stargazersCount;
  final String? language;
  final Map<String, int>? languages;
  final String? readmeUrl;
  final List<String> tags;
  final List<RepositoryStat> stats;

  Repository({
    required this.id,
    required this.name,
    required this.fullName,
    required this.ownerLogin,
    required this.ownerAvatarUrl,
    required this.htmlUrl,
    this.description,
    required this.stargazersCount,
    this.language,
    this.languages,
    this.readmeUrl,
    required this.tags,
    required this.stats,
  });

  factory Repository.fromJson(Map<String, dynamic> json) {
    final repoJson = json['repository'];
    final ownerJson = json['owner'];
    final statsList = json['stats'] as List;
    final tagsList = json['tags'] as List?;

    return Repository(
      id: repoJson['id'],
      name: repoJson['name'],
      fullName: repoJson['full_name'],
      ownerLogin: ownerJson['login'],
      ownerAvatarUrl: ownerJson['avatar_url'],
      htmlUrl: repoJson['html_url'],
      description: repoJson['description']['Valid']
          ? repoJson['description']['String']
          : null,
      stargazersCount:
          statsList.isNotEmpty ? statsList[0]['stargazers_count'] : 0,
      language:
          repoJson['language']['Valid'] ? repoJson['language']['String'] : null,
      languages: repoJson['languages'] != null
          ? Map<String, int>.from(repoJson['languages'])
          : null,
      readmeUrl: repoJson['readme_url']['Valid']
          ? repoJson['readme_url']['String']
          : null,
      tags: tagsList?.map((tag) => tag.toString()).toList() ?? [],
      stats: statsList.map((stat) => RepositoryStat.fromJson(stat)).toList(),
    );
  }
}

class RepositoryStat {
  final DateTime eventDate;
  final DateTime eventTime;
  final int stargazersCount;
  final int watchersCount;
  final int forksCount;
  final int openIssuesCount;
  final DateTime pushedAt;
  final double score;

  RepositoryStat({
    required this.eventDate,
    required this.eventTime,
    required this.stargazersCount,
    required this.watchersCount,
    required this.forksCount,
    required this.openIssuesCount,
    required this.pushedAt,
    required this.score,
  });

  factory RepositoryStat.fromJson(Map<String, dynamic> json) {
    return RepositoryStat(
      eventDate: DateTime.parse(json['event_date']),
      eventTime: DateTime.parse(json['event_time']),
      stargazersCount: json['stargazers_count'],
      watchersCount: json['watchers_count'],
      forksCount: json['forks_count'],
      openIssuesCount: json['open_issues_count'],
      pushedAt: DateTime.parse(json['pushed_at']),
      score: (json['score'] as num).toDouble(),
    );
  }
}
