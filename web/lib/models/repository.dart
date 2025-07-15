class Owner {
  final int id;
  final String login;
  final String avatarUrl;

  Owner({required this.id, required this.login, required this.avatarUrl});

  factory Owner.fromMap(Map<String, dynamic> map) {
    return Owner(
      id: map['id'] ?? 0,
      login: map['login'] ?? '',
      avatarUrl: map['avatar_url'] ?? '',
    );
  }
}

class Repository {
  final String id;
  final String name;
  final String fullName;
  final String? description;
  final String htmlUrl;
  final String? language;
  final int stargazersCount;
  final int forksCount;
  final String createdAt;
  final String updatedAt;
  final bool isPinned;
  final Owner owner;
  final String fetchedAt;
  final String? readmeSnippet;
  final String avatarUrl;
  final String stargazersUrl;
  final String forksUrl;
  final String defaultBranch;

  Repository({
    required this.id,
    required this.name,
    required this.fullName,
    this.description,
    required this.htmlUrl,
    this.language,
    required this.stargazersCount,
    required this.forksCount,
    required this.createdAt,
    required this.updatedAt,
    required this.isPinned,
    required this.owner,
    required this.fetchedAt,
    this.readmeSnippet,
    required this.avatarUrl,
    required this.stargazersUrl,
    required this.forksUrl,
    required this.defaultBranch,
  });

  factory Repository.fromMap(Map<String, dynamic> map) {
    final repo = map['repository'];
    final ownerData = map['owner'];
    final stats = map['stats'] as List<dynamic>?;

    return Repository(
      id: repo['id'].toString(),
      name: repo['name'] ?? '',
      fullName: repo['full_name'] ?? '',
      description: repo['description']?['String'],
      htmlUrl: repo['html_url'] ?? '',
      language: repo['language']?['String'],
      stargazersCount: stats != null && stats.isNotEmpty ? stats[0]['stargazers_count'] ?? 0 : 0,
      forksCount: stats != null && stats.isNotEmpty ? stats[0]['forks_count'] ?? 0 : 0,
      createdAt: repo['created_at'] ?? '',
      updatedAt: repo['updated_at'] ?? '',
      isPinned: repo['is_pinned'] ?? false,
      owner: Owner.fromMap(ownerData ?? {}),
      fetchedAt: repo['last_crawled_at'] ?? '',
      readmeSnippet: null, // This will be fetched later
      avatarUrl: ownerData?['avatar_url'] ?? '',
      stargazersUrl: '${repo['html_url']}/stargazers',
      forksUrl: '${repo['html_url']}/forks',
      defaultBranch: repo['default_branch'] ?? 'main',
    );
  }

  Repository copyWith({
    String? id,
    String? name,
    String? fullName,
    String? description,
    String? htmlUrl,
    String? language,
    int? stargazersCount,
    int? forksCount,
    String? createdAt,
    String? updatedAt,
    bool? isPinned,
    Owner? owner,
    String? fetchedAt,
    String? readmeSnippet,
    String? avatarUrl,
    String? stargazersUrl,
    String? forksUrl,
    String? defaultBranch,
  }) {
    return Repository(
      id: id ?? this.id,
      name: name ?? this.name,
      fullName: fullName ?? this.fullName,
      description: description ?? this.description,
      htmlUrl: htmlUrl ?? this.htmlUrl,
      language: language ?? this.language,
      stargazersCount: stargazersCount ?? this.stargazersCount,
      forksCount: forksCount ?? this.forksCount,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      isPinned: isPinned ?? this.isPinned,
      owner: owner ?? this.owner,
      fetchedAt: fetchedAt ?? this.fetchedAt,
      readmeSnippet: readmeSnippet ?? this.readmeSnippet,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      stargazersUrl: stargazersUrl ?? this.stargazersUrl,
      forksUrl: forksUrl ?? this.forksUrl,
      defaultBranch: defaultBranch ?? this.defaultBranch,
    );
  }
}
