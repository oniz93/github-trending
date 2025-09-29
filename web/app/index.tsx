import React, { useEffect, useState, useRef, useCallback } from 'react';
import { View, FlatList, ActivityIndicator, Dimensions, Text, TouchableOpacity, SafeAreaView, Share, Platform } from 'react-native';
import * as Clipboard from 'expo-clipboard';
import { getRepositories, fetchReadme, trackOpenRepository } from '../services/api';
import { FeedProject } from '../types/repository';
import RepositoryCard from '../components/RepositoryCard';
import SpecialMessageCard from '../components/SpecialMessageCard';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { LinearGradient } from 'expo-linear-gradient';
import { feedStyles } from '../styles/feed';
import ProgressBar from '../components/ProgressBar';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import Toast from '../components/Toast';

const { height: windowHeight } = Dimensions.get('window');

const FeedScreen = () => {
  const [projects, setProjects] = useState<FeedProject[]>([
    {
      id: '-1',
      name: 'Welcome to GitFinder BETA',
      description: 'Discover amazing open source projects in a new way.',
      type: 'message',
    } as FeedProject,
  ]);
  const [loading, setLoading] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [showApology, setShowApology] = useState(false);
  const [scrolledToInitial, setScrolledToInitial] = useState(false);
  const [page, setPage] = useState(0);
  const [sessionId, setSessionId] = useState<string | undefined>(undefined);
  const [toast, setToast] = useState<{ visible: boolean, message: string }>({ visible: false, message: '' });
  const flatListRef = useRef<FlatList>(null);
  const fetchedReadmes = useRef(new Set<string>());
  const projectsRef = useRef(projects);
  const apologyTimerRef = useRef<NodeJS.Timeout | null>(null);
  projectsRef.current = projects;
  const insets = useSafeAreaInsets();
  const height = windowHeight - insets.top - insets.bottom;

  const prefetchReadmes = useCallback(async (startIndex: number, count: number) => {
    for (let i = startIndex; i < startIndex + count && i < projectsRef.current.length; i++) {
      const project = projectsRef.current[i];
      if (project.type === 'repo' && !fetchedReadmes.current.has(project.id)) {
        fetchedReadmes.current.add(project.id);
        try {
          const readme = await fetchReadme(project.id);
          setProjects(prev => prev.map(p => p.id === project.id ? { ...p, readmeSnippet: readme } : p));
        } catch (error) {
          console.error("Failed to fetch readme for", project.full_name, error);
          setProjects(prev => prev.map(p => p.id === project.id ? { ...p, readmeSnippet: "Readme not available at this moment" } : p));
        }
      }
    }
  }, []);

  useEffect(() => {
    const loadSession = async () => {
      const storedSessionId = await AsyncStorage.getItem('sessionId');
      if (storedSessionId) {
        setSessionId(storedSessionId);
      }
      apologyTimerRef.current = setTimeout(() => {
        setShowApology(true);
      }, 10000);
      await loadRepositories(0, storedSessionId || '');
      setInitialLoading(false);
      if (apologyTimerRef.current) {
        clearTimeout(apologyTimerRef.current);
      }
    };
    loadSession();
  }, []);

  useEffect(() => {
    if (!initialLoading && projects.length > 1 && !scrolledToInitial) {
      flatListRef.current?.scrollToIndex({ animated: true, index: 1 });
      setScrolledToInitial(true);
    }
  }, [initialLoading, projects, scrolledToInitial]);

  useEffect(() => {
    if (page === 1 && projects.length > 1) { // After first page is loaded
      prefetchReadmes(1, 5);
    }
  }, [page, projects, prefetchReadmes]);

  const loadRepositories = async (currentPage: number, currentSessionId: string) => {
    if (loading) return;
    setLoading(true);
    try {
      const { repositories: newRepositories, sessionId: newSessionId } = await getRepositories(currentPage, currentSessionId);
      
      setProjects(prev => [...prev, ...newRepositories]);

      if (newSessionId) {
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

  const shareProject = async (project: FeedProject) => {
    if (sessionId) {
      trackOpenRepository(sessionId, project.id);
    }
    if (Platform.OS === 'web') {
      try {
        await Clipboard.setStringAsync(project.html_url);
        setToast({ visible: true, message: 'Link copied to clipboard!' });
      } catch (e) {
        setToast({ visible: true, message: 'Failed to copy link.' });
      }
    } else {
      try {
        await Share.share({
          message: `Check out this repository on GitHub: ${project.html_url}`,
          url: project.html_url,
          title: project.full_name,
        });
      } catch (error) {
        console.error('Error sharing:', error);
      }
    }
  };

  const onViewableItemsChanged = useRef(({ viewableItems }: { viewableItems: any[] }) => {
    if (viewableItems.length > 0) {
      const lastVisibleItem = viewableItems[viewableItems.length - 1];
      const nextIndex = lastVisibleItem.index + 1;
      if (nextIndex < projectsRef.current.length) {
        prefetchReadmes(nextIndex, 1);
      }
    }
  }).current;

  const renderItem = ({ item }: { item: FeedProject }) => {
    const card = item.type === 'message'
      ? <SpecialMessageCard project={item} />
      : <RepositoryCard
          project={item}
          sessionId={sessionId}
          renderMarkdown={(content) => content} // This will be replaced with a proper markdown renderer
          formatNumber={(num) => num.toString()} // Placeholder
          shareProject={shareProject}
        />;

    return (
      <View style={{ height, width: '100%' }}>
        {card}
      </View>
    );
  };

  return (
    <SafeAreaView style={feedStyles.container}>
      <LinearGradient
        colors={['#1e2a3a', '#12161f', '#0d1117']}
        style={feedStyles.gradient}
      />
      <View style={feedStyles.topBar}>
        <TouchableOpacity style={feedStyles.topBarButton}>
          <Text style={feedStyles.topBarButtonText}>Login</Text>
        </TouchableOpacity>
      </View>
      {initialLoading && (
        <View style={feedStyles.loadingContainer}>
          <ProgressBar duration={10000} onFinish={() => {}} />
          {showApology && <Text style={feedStyles.apologyText}>Sorry, this is taking longer than expected...</Text>}
        </View>
      )}
      <FlatList
        ref={flatListRef}
        data={projects}
        renderItem={renderItem}
        keyExtractor={(item) => item.id.toString()}
        pagingEnabled
        showsVerticalScrollIndicator={false}
        onEndReached={() => loadRepositories(page, sessionId || '')}
                onEndReachedThreshold={0.9}
        ListFooterComponent={loading && !initialLoading ? <ActivityIndicator size="large" color="#ffffff" /> : null}
        getItemLayout={(data, index) => ({ length: height, offset: height * index, index })}
        onViewableItemsChanged={onViewableItemsChanged}
        viewabilityConfig={{
          itemVisiblePercentThreshold: 50
        }}
        snapToAlignment="start"
        decelerationRate={"fast"}
        snapToInterval={height}
      />
      <Toast
        visible={toast.visible}
        message={toast.message}
        onDismiss={() => setToast({ ...toast, visible: false })}
      />
    </SafeAreaView>
  );
};

export default FeedScreen;

