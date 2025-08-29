import React from 'react';
import { View, Text, StyleSheet, Image, TouchableOpacity } from 'react-native';
import { FeedProject } from '../types/repository';

interface RepoCardProps {
  project: FeedProject;
  renderMarkdown: (content: string) => string;
  formatNumber: (num: number) => string;
  shareProject: (project: FeedProject) => void;
}

const languageColors: { [key: string]: string } = {
  TypeScript: '#3178c6',
  JavaScript: '#f1e05a',
  Python: '#3572A5',
  Java: '#b07219',
  Ruby: '#701516',
  Go: '#00ADD8',
  Rust: '#dea584',
  C: '#555555',
  'C++': '#f34b7d',
  'C#': '#178600',
  PHP: '#4F5D95',
  Swift: '#ffac45',
  Kotlin: '#A97BFF',
  Dart: '#00B4AB',
  Shell: '#89e051',
  HTML: '#e34c26',
  CSS: '#563d7c',
  Vue: '#41b883',
  Svelte: '#ff3e00',
  React: '#61dafb'
};

const getLanguageColor = (language: string | null): string => {
  if (!language) return '#808080';
  return languageColors[language] || '#808080';
};

const RepoCard: React.FC<RepoCardProps> = ({ project, renderMarkdown, formatNumber, shareProject }) => {
  return (
    <View style={styles.cardContainer}>
      <View style={styles.header}>
        <Text style={styles.title}>{project.name}</Text>
        {project.language && (
          <View style={styles.languageContainer}>
            <View style={[styles.languageDot, { backgroundColor: getLanguageColor(project.language) }]} />
            <Text style={styles.languageText}>{project.language}</Text>
          </View>
        )}
      </View>
      <Text style={styles.description}>{project.description}</Text>

      <View style={styles.readmeContainer}>
        {project.readmeSnippet ? (
          <View style={{ flex: 1 }} dangerouslySetInnerHTML={{ __html: renderMarkdown(project.readmeSnippet) }} />
        ) : (
          <Text style={styles.loadingText}>Loading README...</Text>
        )}
      </View>

      <View style={styles.statsContainer}>
        <TouchableOpacity style={styles.statItem} onPress={() => {}}>
          <Text style={styles.statIcon}>★</Text>
          <Text style={styles.statText}>{formatNumber(project.stargazers_count)}</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.statItem} onPress={() => {}}>
          <Text style={styles.statIcon}>⑂</Text>
          <Text style={styles.statText}>{formatNumber(project.forks_count)}</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.statItem} onPress={() => shareProject(project)}>
          <Text style={styles.statIcon}>🔗</Text>
          <Text style={styles.statText}>Share</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.footer}>
        <Image source={{ uri: project.owner.avatar_url }} style={styles.avatar} />
        <Text style={styles.ownerLogin}>{project.owner.login}</Text>
        <TouchableOpacity style={styles.viewButton} onPress={() => {}}>
          <Text style={styles.viewButtonText}>View</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  cardContainer: {
    flex: 1,
    padding: 16,
    backgroundColor: 'transparent',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  title: {
    fontFamily: 'serif',
    fontSize: 24,
    color: 'white',
    marginRight: 8,
  },
  languageContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  languageDot: {
    width: 12,
    height: 12,
    borderRadius: 6,
    marginRight: 4,
  },
  languageText: {
    fontFamily: 'monospace',
    fontSize: 14,
    color: '#8b949e',
  },
  description: {
    fontFamily: 'serif',
    fontSize: 16,
    color: '#e1e4e8',
    marginBottom: 16,
  },
  readmeContainer: {
    flex: 1,
    overflow: 'hidden',
    borderRadius: 8,
    backgroundColor: 'rgba(22, 27, 34, 0.5)',
    padding: 16,
    backdropFilter: 'blur(10px)',
  },
  loadingText: {
    color: '#8b949e',
  },
  statsContainer: {
    position: 'absolute',
    right: 16,
    bottom: 120, 
    alignItems: 'center',
    gap: 24,
  },
  statItem: {
    alignItems: 'center',
  },
  statIcon: {
    fontSize: 24,
    color: '#f0a500',
  },
  statText: {
    color: 'white',
    fontSize: 14,
    fontFamily: 'monospace',
    marginTop: 4,
  },
  footer: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 16,
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    marginRight: 8,
  },
  ownerLogin: {
    fontFamily: 'monospace',
    fontSize: 16,
    color: 'white',
    flex: 1,
  },
  viewButton: {
    paddingVertical: 8,
    paddingHorizontal: 16,
    backgroundColor: 'rgba(255, 255, 255, 0.1)',
    borderRadius: 6,
  },
  viewButtonText: {
    color: 'white',
    fontFamily: 'monospace',
  },
});

export default RepoCard;
