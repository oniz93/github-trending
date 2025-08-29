import React from 'react';
import { View, Text, StyleSheet, Image, TouchableOpacity } from 'react-native';
import { FeedProject } from '../types/repository';

interface PromotedRepoCardProps {
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

const PromotedRepoCard: React.FC<PromotedRepoCardProps> = ({ project, renderMarkdown, formatNumber, shareProject }) => {
  return (
    <View style={styles.cardContainer}>
      <View style={styles.glow} />
      <View style={styles.innerContainer}>
        <View style={styles.header}>
            <View style={styles.featuredBadge}>
                <Text style={styles.featuredIcon}>✨</Text>
                <Text style={styles.featuredText}>Featured</Text>
            </View>
            <TouchableOpacity style={styles.starUsButton}>
                <Text style={styles.starUsIcon}>★</Text>
                <Text style={styles.starUsText}>Star us to get featured!</Text>
            </TouchableOpacity>
        </View>
        <Text style={styles.title}>{project.name}</Text>
        <Text style={styles.description}>{project.description}</Text>

        <View style={styles.readmeContainer}>
          {project.readmeSnippet ? (
            <View style={{ flex: 1 }} dangerouslySetInnerHTML={{ __html: renderMarkdown(project.readmeSnippet) }} />
          ) : (
            <Text style={styles.loadingText}>Loading README...</Text>
          )}
        </View>

        <View style={styles.statsContainer}>
            <TouchableOpacity style={styles.statItem}>
                <Text style={styles.statIcon}>★</Text>
                <Text style={styles.statText}>{formatNumber(project.stargazers_count)}</Text>
            </TouchableOpacity>
            <TouchableOpacity style={styles.statItem}>
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
          <TouchableOpacity style={styles.viewButton}>
            <Text style={styles.viewButtonText}>View Project</Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
    cardContainer: {
        flex: 1,
        borderRadius: 12,
        overflow: 'hidden',
      },
      glow: {
        position: 'absolute',
        top: -2, left: -2, right: -2, bottom: -2,
        borderRadius: 14,
        backgroundColor: 'purple',
        opacity: 0.5,
        filter: 'blur(10px)',
      },
      innerContainer: {
        flex: 1,
        backgroundColor: 'rgba(13, 17, 23, 0.9)',
        borderRadius: 12,
        padding: 16,
        overflow: 'hidden',
      },
      header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 16,
      },
      featuredBadge: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'purple',
        paddingVertical: 4,
        paddingHorizontal: 8,
        borderRadius: 12,
      },
      featuredIcon: {
        fontSize: 12,
        marginRight: 4,
      },
      featuredText: {
        color: 'white',
        fontWeight: 'bold',
        fontSize: 12,
      },
      starUsButton: {
        flexDirection: 'row',
        alignItems: 'center',
        backgroundColor: 'rgba(255, 255, 255, 0.1)',
        paddingVertical: 6,
        paddingHorizontal: 12,
        borderRadius: 16,
      },
      starUsIcon: {
        color: '#f0a500',
        marginRight: 4,
      },
      starUsText: {
        color: 'white',
        fontSize: 12,
      },
      title: {
        fontFamily: 'serif',
        fontSize: 28,
        color: 'white',
        fontWeight: 'bold',
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
        backgroundColor: 'rgba(22, 27, 34, 0.7)',
        padding: 16,
        borderWidth: 1,
        borderColor: 'rgba(255,255,255,0.1)',
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
        backgroundColor: 'rgba(0,0,0,0.5)',
        padding: 8,
        borderRadius: 20,
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
        borderWidth: 1,
        borderColor: 'purple',
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
        backgroundColor: 'purple',
        borderRadius: 6,
      },
      viewButtonText: {
        color: 'white',
        fontFamily: 'monospace',
        fontWeight: 'bold',
      },
});

export default PromotedRepoCard;
