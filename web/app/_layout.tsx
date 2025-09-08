import { Stack } from "expo-router";

export default function RootLayout() {
  return (
    <>
      <Stack screenOptions={{ headerShown: false }}>
        <Stack.Screen name="index" />
        <Stack.Screen name="readme" options={{ headerShown: true, title: "README", headerBackTitle: "Back", headerStyle: { backgroundColor: '#0d1117' }, headerTintColor: 'white' }} />
      </Stack>
    </>
  );
}