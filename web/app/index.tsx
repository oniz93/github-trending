import React, { useEffect, useState, useRef } from 'react';
import { View, FlatList, ActivityIndicator, StyleSheet, Dimensions, Text, TouchableOpacity } from 'react-native';
import { getRepositories, fetchReadme } from '../services/api';
import { FeedProject } from '../types/repository';
import RepositoryCard from '../components/RepositoryCard';
import PromotedRepoCard from '../components/PromotedRepositoryCard';
import SpecialMessageCard from '../components/SpecialMessageCard';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { LinearGradient } from 'expo-linear-gradient';
import { feedStyles } from '../styles/feed';

const { height } = Dimensions.get('window');

const FeedScreen = () => {
  const [projects, setProjects] = useState<FeedProject[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(0);
  const [sessionId, setSessionId] = useState<string | undefined>(undefined);
  const flatListRef = useRef<FlatList>(null);

  useEffect(() => {
    const loadSession = async () => {
      const storedSessionId = await AsyncStorage.getItem('sessionId');
      if (storedSessionId) {
        setSessionId(storedSessionId);
      }
      loadRepositories(0, storedSessionId || undefined);
    };
    loadSession();
  }, []);

  const loadRepositories = async (currentPage: number, currentSessionId?: string) => {
    if (loading) return;
    setLoading(true);
    try {
      const { repositories: newRepositories, sessionId: newSessionId } = await getRepositories(currentPage, currentSessionId);
      
      let projectsWithMessages: FeedProject[] = [...newRepositories];
      if (currentPage === 0) {
        projectsWithMessages.unshift({
            id: '-1',
            name: 'Welcome to GitTok',
            description: 'Discover amazing open source projects in a new way.',
            type: 'message',
        } as FeedProject);
      }

      setProjects(prev => [...prev, ...projectsWithMessages]);

      if (newSessionId && newSessionId !== currentSessionId) {
        setSessionId(newSessionId);
        await AsyncStorage.setItem('sessionId', newSessionId);
      }
      setPage(currentPage + 1);
    } catch (error) {
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const onViewableItemsChanged = useRef(({ viewableItems }: { viewableItems: any[] }) => {
    viewableItems.forEach(async (item) => {
      const project = item.item as FeedProject;
      if (project.type === 'repo' && !project.readmeSnippet) {
        const readme = await fetchReadme(project.full_name);
        setProjects(prev => prev.map(p => p.id === project.id ? { ...p, readmeSnippet: readme } : p));
      }
    });
  }).current;

  const renderItem = ({ item }: { item: FeedProject }) => {
    if (item.type === 'message') {
      return <SpecialMessageCard project={item} />;
    }
    // For now, we'll use the standard card for all repos.
    // We can add logic here to use PromotedRepoCard for specific projects.
    return (
      <RepositoryCard 
        project={item} 
        renderMarkdown={(content) => content} // This will be replaced with a proper markdown renderer
        formatNumber={(num) => num.toString()} // Placeholder
        shareProject={(p) => console.log('share', p.name)} // Placeholder
      />
    );
  };

  return (
    <View style={feedStyles.container}>
      <LinearGradient
        colors={['#1e2a3a', '#12161f', '#0d1117']}
        style={feedStyles.gradient}
      />
      <View style={feedStyles.topBar}>
        <TouchableOpacity style={feedStyles.topBarButton}>
          <Text style={feedStyles.topBarButtonText}>Login</Text>
        </TouchableOpacity>
      </View>
      <FlatList
        ref={flatListRef}
        data={projects}
        renderItem={renderItem}
        keyExtractor={(item) => item.id.toString()}
        pagingEnabled
        showsVerticalScrollIndicator={false}
        onEndReached={() => loadRepositories(page, sessionId)}
        onEndReachedThreshold={0.5}
        ListFooterComponent={loading ? <ActivityIndicator size="large" color="#ffffff" /> : null}
        getItemLayout={(data, index) => ({ length: height, offset: height * index, index })}
        onViewableItemsChanged={onViewableItemsChanged}
        viewabilityConfig={{
          itemVisiblePercentThreshold: 50
        }}
        snapToAlignment="start"
        decelerationRate={"fast"}
        snapToInterval={height}
      />
    </View>
  );
};

export default FeedScreen;