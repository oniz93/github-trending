import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, Image, Dimensions, Platform, TouchableOpacity } from 'react-native';
import { useRouter } from 'expo-router';
import { Repository } from '../types/repository';
import { fetchReadme } from '../services/api';
import { WebView } from 'react-native-webview';
import { Gesture, GestureDetector } from 'react-native-gesture-handler';

interface RepositoryCardProps {
  repository: Repository;
}

const { height } = Dimensions.get('window');

const RepositoryCard: React.FC<RepositoryCardProps> = ({ repository }) => {
  const router = useRouter();
  const [readme, setReadme] = useState('');

  useEffect(() => {
    const loadReadme = async () => {
      if (repository.full_name) {
        const readmeContent = await fetchReadme(repository.full_name);
        setReadme(readmeContent);
      }
    };

    loadReadme();
  }, [repository.full_name]);

  const fling = Gesture.Fling()
    .direction(1)
    .onEnd(() => {
        router.push({ pathname: '/readme', params: { readmeUrl: repository.readme_url } });
    });

  const renderReadme = () => {
    const styles = `
      body { 
        background-color: #0d1117; 
        color: white; 
        font-family: sans-serif; 
        margin: 0;
        padding: 16px;
      }
      img { 
        max-width: 100%; 
        height: auto; 
        border-radius: 6px;
      } 
      a { color: #58a6ff; }
      pre { 
        background-color: #161b22; 
        padding: 16px; 
        border-radius: 6px; 
        overflow: auto;
      }
      code {
        font-family: monospace;
      }
    `;

    if (Platform.OS === 'web') {
      return (
        <iframe
          srcDoc={`<style>${styles}</style>${readme}`}
          style={{ width: '100%', height: '100%', border: 'none' }}
        />
      );
    } else {
      return (
        <WebView
            originWhitelist={['*']}
            source={{ html: `<style>${styles}</style>${readme}` }}
            style={{backgroundColor: '#0d1117'}}
            scrollEnabled={false}
        />
      )
    }
  }

  return (
    <GestureDetector gesture={fling}>
      <View style={styles.container}>
        <View style={styles.infoContainer}>
            <View style={styles.header}>
              <Image source={{ uri: repository.owner.avatar_url }} style={styles.avatar} />
              <Text style={styles.owner}>{repository.owner.login}</Text>
            </View>
            <Text style={styles.title}>{repository.name}</Text>
            <Text style={styles.description}>{repository.description}</Text>
        </View>
        <View style={styles.readmeContainer}>
          {renderReadme()}
        </View>
        <View style={styles.footer}>
            <TouchableOpacity onPress={() => router.push({ pathname: '/readme', params: { readmeUrl: repository.readme_url } })}>
                <Text style={styles.showMore}>Show More</Text>
            </TouchableOpacity>
        </View>
        <View style={styles.statsContainer}>
            <View style={styles.stat}>
                <Text style={styles.statIcon}>★</Text>
                <Text style={styles.statText}>{repository.stargazers_count}</Text>
            </View>
            <View style={styles.stat}>
                <Text style={styles.statIcon}>●</Text>
                <Text style={styles.statText}>{repository.language}</Text>
            </View>
        </View>
      </View>
    </GestureDetector>
  );
};

const styles = StyleSheet.create({
  container: {
    height: height,
    backgroundColor: '#0d1117',
    padding: 16,
    flex: 1,
    justifyContent: 'space-between',
  },
  infoContainer: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 16,
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    marginRight: 8,
  },
  owner: {
    color: 'white',
    fontSize: 16,
    fontFamily: 'monospace',
  },
  title: {
    color: 'white',
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 8,
  },
  description: {
    color: '#8b949e',
    fontSize: 16,
    fontFamily: 'serif',
  },
  readmeContainer: {
    height: 300,
    overflow: 'hidden',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#30363d',
  },
  footer: {
    paddingVertical: 16,
    alignItems: 'center',
  },
  showMore: {
    color: '#58a6ff',
    fontSize: 16,
  },
  statsContainer: {
    position: 'absolute',
    right: 16,
    top: '50%',
    transform: [{ translateY: -50 }],
    alignItems: 'center',
  },
  stat: {
    alignItems: 'center',
    marginBottom: 16,
  },
  statIcon: {
    fontSize: 24,
    color: '#f0a500',
  },
  statText: {
    color: 'white',
    fontSize: 14,
    fontFamily: 'monospace',
  },
});

export default RepositoryCard;
