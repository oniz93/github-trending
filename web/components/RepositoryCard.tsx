import React, { useState } from 'react';
import { View, Text, StyleSheet, Image, TouchableOpacity, Platform, Linking } from 'react-native';
import { FeedProject } from '../types/repository';
import { WebView } from 'react-native-webview';
import Tooltip from './Tooltip';
import { trackOpenRepository } from '../services/api';
import AutoShrinkingText from './AutoShrinkingText';

interface RepoCardProps {
  project: FeedProject;
  sessionId: string | undefined;
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

const RepoCard: React.FC<RepoCardProps> = ({ project, sessionId, renderMarkdown, formatNumber, shareProject }) => {
  const [visibleTooltip, setVisibleTooltip] = useState<string | null>(null);
  const totalLanguagesSize = Object.values(project.languages).reduce((acc, size) => acc + size, 0);

  const statItems = [
    { key: 'stars', icon: '★', value: formatNumber(project.stargazers_count), label: 'Stars' },
    { key: 'forks', icon: '⑂', value: formatNumber(project.forks_count), label: 'Forks' },
    { key: 'watchers', icon: '👁', value: formatNumber(project.watchers_count), label: 'Watchers' },
    { key: 'issues', icon: '🐞', value: formatNumber(project.open_issues_count), label: 'Open Issues' },
    { key: 'share', icon: '🔗', value: 'Share', label: 'Share' },
  ];

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

      <View style={styles.statsContainer}>
        {statItems.map(item => (
          <View
            key={item.key}
            onMouseEnter={() => Platform.OS === 'web' && setVisibleTooltip(item.key)}
            onMouseLeave={() => Platform.OS === 'web' && setVisibleTooltip(null)}
          >
            <TouchableOpacity
              style={styles.statItem}
              onPress={() => item.key === 'share' && shareProject(project)}
              onLongPress={() => Platform.OS !== 'web' && setVisibleTooltip(item.key)}
              onPressOut={() => Platform.OS !== 'web' && setVisibleTooltip(null)}
            >
              {visibleTooltip === item.key && <Tooltip text={item.label} />}
              <Text style={styles.statIcon}>{item.icon}</Text>
              <Text style={styles.statText}>{item.value}</Text>
            </TouchableOpacity>
          </View>
        ))}
      </View>

      <Text style={styles.description}>{project.description}</Text>

      {project.topics && project.topics.length > 0 && (
        <View style={styles.topicsContainer}>
          {project.topics.slice(0, 5).map(topic => (
            <View key={topic} style={styles.topicBadge}>
              <Text style={styles.topicText}>{topic}</Text>
            </View>
          ))}
        </View>
      )}

      {totalLanguagesSize > 0 && (
        <View style={styles.languagesBar}>
          {Object.entries(project.languages).map(([lang, size]) => (
            <View key={lang} style={{ width: `${(size / totalLanguagesSize) * 100}%`, backgroundColor: getLanguageColor(lang), height: 8 }} />
          ))}
        </View>
      )}

      <View style={styles.readmeContainer}>
        {project.readmeSnippet ? (
          Platform.OS === 'web' ? (
            <iframe
              srcDoc={`
                <style>
                  body {
                    background-color: transparent;
                    color: #c9d1d9;
                    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";
                    font-size: 16px;
                    line-height: 1.5;
                    margin: 0;
                    padding: 16px;
                  }
                  a {
                    color: #58a6ff;
                  }
                  h1, h2, h3, h4, h5, h6 {
                    color: #c9d1d9;
                    border-bottom-color: #30363d;
                  }
                  pre, code {
                    background-color: #161b22;
                    border-color: #30363d;
                  }
                </style>
                ${project.readmeSnippet}
              `}
              style={{ width: '100%', height: '100%', border: 'none' }}
              sandbox="allow-scripts"
              scrolling="no"
            />
          ) : (
            <WebView
              originWhitelist={['*']}
              source={{ html: `
                <style>
                  body {
                    background-color: transparent;
                    color: #c9d1d9;
                    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji";
                    font-size: 16px;
                    line-height: 1.5;
                    padding: 16px;
                  }
                  a {
                    color: #58a6ff;
                  }
                  h1, h2, h3, h4, h5, h6 {
                    color: #c9d1d9;
                    border-bottom-color: #30363d;
                  }
                  pre, code {
                    background-color: #161b22;
                    border-color: #30363d;
                  }
                </style>
                ${project.readmeSnippet}
              ` }}
              style={{ backgroundColor: 'transparent', flex: 1 }}
              scrollEnabled={false}
            />
          )
        ) : (
          <Text style={styles.loadingText}>Loading README...</Text>
        )}
      </View>

      <View style={styles.footer}>
        <Image source={{ uri: project.owner.avatar_url }} style={styles.avatar} />
        <View style={styles.footerTextContainer}>
          <Text style={styles.ownerLogin}>{project.owner.login}</Text>
          <View style={styles.footerMetaContainer}>
            <AutoShrinkingText
              text={`${project.license ? project.license.name + ' | ' : ''}Pushed ${new Date(project.pushed_at).toLocaleDateString()}`}
              style={styles.pushedAtText}
            />
          </View>
        </View>
          <TouchableOpacity style={styles.viewButton} onPress={() => {
              if (sessionId) {
                  trackOpenRepository(sessionId, project.id);
              }
              Linking.openURL(project.html_url);
          }}>
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
    backdropFilter: 'blur(10px)',
  },
  loadingText: {
    color: '#8b949e',
  },
  statsContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 16,
    marginBottom: 8,
  },
  statItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  statIcon: {
    fontSize: 16,
    color: '#f0a500',
  },
  statText: {
    color: 'white',
    fontSize: 14,
    fontFamily: 'monospace',
  },
  topicsContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 8,
  },
  topicBadge: {
    backgroundColor: 'rgba(33, 150, 243, 0.2)',
    borderRadius: 16,
    paddingVertical: 4,
    paddingHorizontal: 12,
  },
  topicText: {
    color: '#58a6ff',
    fontSize: 12,
    fontWeight: 'bold',
  },
  languagesBar: {
    flexDirection: 'row',
    height: 8,
    borderRadius: 4,
    overflow: 'hidden',
    marginBottom: 16,
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
  footerTextContainer: {
    flex: 1,
  },
  ownerLogin: {
    fontFamily: 'monospace',
    fontSize: 16,
    color: 'white',
  },
  footerMetaContainer: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  licenseText: {
    fontFamily: 'monospace',
    fontSize: 14,
    color: '#8b949e',
    marginRight: 8,
  },
  pushedAtText: {
    fontFamily: 'monospace',
    fontSize: 14,
    color: '#8b949e',
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
