import React, { useEffect, useState } from 'react';
import { View, Text, StyleSheet, Image, Dimensions, Platform } from 'react-native';
import { Repository } from '../types/repository';
import { fetchReadme } from '../services/api';
import { WebView } from 'react-native-webview';

interface RepositoryCardProps {
  repository: Repository;
}

const { height } = Dimensions.get('window');

const RepositoryCard: React.FC<RepositoryCardProps> = ({ repository }) => {
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
      <View style={styles.container}>
        <View style={styles.infoContainer}>
            <View style={styles.header}>
              <Image source={{ uri: repository.owner.avatar_url }} style={styles.avatar} />
              <Text style={styles.owner}>{repository.owner.login}</Text>
            </View>
            <Text style={styles.title}>{repository.name}</Text>
            <Text style={styles.description}>{repository.description}</Text>
            <View style={styles.footer}>
              <View style={styles.statsContainer}>
                <Text style={styles.statsText}>★ {repository.stargazers_count}</Text>
                <Text style={styles.statsText}>● {repository.language}</Text>
              </View>
            </View>
        </View>
        <View style={styles.readmeContainer}>
          {renderReadme()}
        </View>
      </View>
  );
};

const styles = StyleSheet.create({
  container: {
    height: height,
    backgroundColor: '#0d1117',
    padding: 16,
    flex: 1,
    flexDirection: 'column',
  },
  infoContainer: {
    // This will take up as much space as it needs
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
    marginBottom: 16,
  },
  readmeContainer: {
    flex: 1, // This will take up the remaining space
    overflow: 'hidden',
    marginTop: 16,
  },
  footer: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 16,
  },
  statsContainer: {
    flexDirection: 'row',
  },
  statsText: {
    color: 'white',
    marginRight: 16,
  },
});

export default RepositoryCard;
