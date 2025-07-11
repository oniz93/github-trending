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
  });

  factory Repository.fromJson(Map<String, dynamic> json) {
    print('Parsing repository JSON: $json');
    final id = json['id'];
    final stargazersCount = json['stargazers_count'];

    print('id type: ${id.runtimeType}, value: $id');
    print('stargazers_count type: ${stargazersCount.runtimeType}, value: $stargazersCount');

    return Repository(
      id: id,
      name: json['name'],
      fullName: json['full_name'],
      ownerLogin: json['owner']['login'],
      ownerAvatarUrl: json['owner']['avatar_url'],
      htmlUrl: json['html_url'],
      description: json['description']['Valid'] ? json['description']['String'] : null,
      stargazersCount: stargazersCount,
      language: json['language']['Valid'] ? json['language']['String'] : null,
      languages: json['languages'] != null
          ? Map<String, int>.from(json['languages'])
          : null,
      readmeUrl: json['readme_url']['Valid'] ? json['readme_url']['String'] : null,
    );
  }
}