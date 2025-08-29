import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { View, StyleSheet, ActivityIndicator, Platform } from 'react-native';
import { WebView } from 'react-native-webview';
import { fetchReadme } from '../services/api';

const ReadmeScreen = () => {
  const { readmeUrl } = useLocalSearchParams();
  const [readme, setReadme] = React.useState('');
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    const loadReadme = async () => {
      if (typeof readmeUrl === 'string') {
        const readmeContent = await fetchReadme(readmeUrl);
        setReadme(readmeContent);
      }
      setLoading(false);
    };

    loadReadme();
  }, [readmeUrl]);

  if (loading) {
    return <ActivityIndicator size="large" color="#ffffff" />;
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
          style={{ width: '100%', height: '100%', border: 'none' }}
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
    <View style={styles.container}>
      {renderReadme()}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0d1117',
  },
  webview: {
    flex: 1,
    backgroundColor: '#0d1117',
  },
});

export default ReadmeScreen;