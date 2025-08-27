import React, { useEffect, useState, useRef } from 'react';
import { View, FlatList, ActivityIndicator, StyleSheet, Dimensions } from 'react-native';
import { getRepositories } from '../services/api';
import { Repository } from '../types/repository';
import RepositoryCard from '../components/RepositoryCard';
import AsyncStorage from '@react-native-async-storage/async-storage';
import TopBar from '../components/TopBar';

const { height } = Dimensions.get('window');

const FeedScreen = () => {
  const [repositories, setRepositories] = useState<Repository[]>([]);
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
      setRepositories(prev => [...prev, ...newRepositories]);
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

  const renderItem = ({ item }: { item: Repository }) => (
    <RepositoryCard repository={item} />
  );

  return (
    <View style={styles.container}>
      <TopBar />
      <FlatList
        ref={flatListRef}
        data={repositories}
        renderItem={renderItem}
        keyExtractor={(item) => item.id.toString()}
        pagingEnabled
        showsVerticalScrollIndicator={false}
        onEndReached={() => loadRepositories(page, sessionId)}
        onEndReachedThreshold={0.5}
        ListFooterComponent={loading ? <ActivityIndicator size="large" color="#ffffff" /> : null}
        getItemLayout={(data, index) => ({ length: height, offset: height * index, index })}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0d1117',
    paddingTop: 80, // Add padding for the top bar
  },
});

export default FeedScreen;
