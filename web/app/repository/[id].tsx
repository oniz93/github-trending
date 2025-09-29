import React, { useEffect, useState } from 'react';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { View, Text, StyleSheet, ActivityIndicator, Platform, ScrollView, Button, Linking } from 'react-native';
import { WebView } from 'react-native-webview';
import { getRepository, fetchReadme } from '../../services/api';
import { FeedProject } from '../../types/repository';

const RepositoryScreen = () => {
  const { id } = useLocalSearchParams();
  const router = useRouter();
  const [repository, setRepository] = useState<FeedProject | null>(null);
  const [readme, setReadme] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadData = async () => {
      if (typeof id !== 'string') return;

      try {
        const repoData = await getRepository(id);
        setRepository(repoData);

        const readmeContent = await fetchReadme(id);
        setReadme(readmeContent);
      } catch (error) {
        console.error('Error loading repository data:', error);
      } finally {
        setLoading(false);
      }
    };

    loadData();
  }, [id]);

  if (loading) {
    return <ActivityIndicator size="large" color="#ffffff" />;
  }

  if (!repository) {
    return (
      <View style={styles.container}>
        <Text style={styles.errorText}>Repository not found.</Text>
        <Button title="Go Back" onPress={() => router.back()} />
      </View>
    );
  }

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
          srcDoc={`<style>${styles}</style><div class="markdown-body">${readme}</div>`}
          style={{ width: '100%', height: '100vh', border: 'none' }}
        />
      );
    } else {
      return (
        <WebView
            originWhitelist={['*']}
            source={{ html: `<style>${styles}</style><div class="markdown-body">${readme}</div>` }}
            style={styles.webview}
        />
      )
    }
  }

  return (
    <ScrollView style={styles.container}>
      <View style={styles.header}>
        <Button title="< Back" onPress={() => router.push('/')} />
        <Text style={styles.title}>{repository.full_name}</Text>
        <Button title="View on GitHub" onPress={() => Linking.openURL(repository.html_url)} />
      </View>
      <View style={styles.readmeContainer}>
        {renderReadme()}
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0d1117',
  },
  header: {
    padding: 16,
    backgroundColor: '#161b22',
    borderBottomWidth: 1,
    borderBottomColor: '#30363d',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  title: {
    color: 'white',
    fontSize: 20,
    fontWeight: 'bold',
  },
  readmeContainer: {
    flex: 1,
  },
  webview: {
    flex: 1,
    backgroundColor: '#0d1117',
  },
  errorText: {
    color: 'white',
    textAlign: 'center',
    marginTop: 20,
  },
});

export default RepositoryScreen;
