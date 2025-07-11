# Frontend Development Plan

This document outlines the plan for building the frontend application for the GitHub Trending project. The goal is to create a simple, mobile-first application that is deployable to the web and can be packaged for mobile app stores (iOS and Android).

## 1. Technology Stack

- **Framework:** **Flutter**.
  - **Why Flutter?** It allows us to build a high-quality, performant application for web, iOS, and Android from a single Dart codebase. Its "write once, run anywhere" nature is ideal for our requirements. It also has excellent support for the kind of gesture-based, full-screen UI we need.
- **State Management:** **Riverpod**.
  - **Why Riverpod?** It is a modern, flexible, and compile-safe state management library for Flutter that makes it easy to manage dependencies (like our API client) and UI state.
- **Key Packages:**
  - `http`: For making API calls to our backend.
  - `flutter_markdown`: To render repository README files.
  - `url_launcher`: To open repository URLs in the browser.
  - `shared_preferences`: To persist the session ID locally.
  - `scrollable_positioned_list`: A more powerful alternative to the default `ListView` that will help with infinite scrolling and managing the list position.

## 2. Project Setup

1.  **Install Flutter:** Follow the official guide to install the Flutter SDK: [https://docs.flutter.dev/get-started/install](https://docs.flutter.dev/get-started/install)
2.  **Create the Project:** Navigate to the `web/` directory and run the following command to create a new Flutter application.
    ```bash
    flutter create .
    ```
3.  **Add Dependencies:** Open the `pubspec.yaml` file and add the necessary packages under the `dependencies` section.
    ```yaml
    dependencies:
      flutter:
        sdk: flutter
      http: ^1.2.1
      flutter_riverpod: ^2.5.1
      flutter_markdown: ^0.7.1
      url_launcher: ^6.2.6
      shared_preferences: ^2.2.3
      scrollable_positioned_list: ^0.3.8
    ```
4.  **Run `flutter pub get`** to install the new dependencies.

## 3. Core Logic & Services

### a. Session Management (`session_provider.dart`)

- Create a Riverpod provider to manage the `sessionId`.
- On app start, it will try to load the `sessionId` and its last-used timestamp from `shared_preferences`.
- If the session is older than 1 hour or doesn't exist, it will be considered invalid.
- It will provide a method to save a new `sessionId` and update the timestamp upon successful API calls.

### b. API Client (`api_service.dart`)

- Create a service class that handles all HTTP requests.
- **`retrieveList()` method:**
  - It will get the current `sessionId` from the `session_provider`.
  - It will make a `GET` request to the `/retrieveList` endpoint, passing the session ID if available.
  - On success, it will parse the list of repositories and update the session provider with the new `sessionId` from the response.
- **`trackOpenRepository()` method:**
  - It will take a `repositoryId` as input.
  - It will make a `POST` request to `/trackOpenRepository`, including the current `sessionId`.

## 4. UI/UX Implementation

The UI will be centered around a full-screen, swipeable view of repositories.

### Step 1: Main View (`repository_list_view.dart`)

- This will be the main screen of the app.
- It will use a `ScrollablePositionedList` to display the repositories. This widget gives us fine-grained control over scrolling, which is perfect for implementing the "load more" feature.
- A `ItemScrollController` will be used to listen to scroll events. When the user scrolls to approximately 90% of the current list's length, the view will trigger the API service to fetch the next page of repositories.
- It will handle the initial loading state (showing a progress indicator) and any potential error states (showing an error message with a retry button).

### Step 2: Repository Card (`repository_card.dart`)

- This widget represents a single, full-screen repository view inside the list.
- It will receive a `Repository` object.
- The main content area will use the `flutter_markdown` widget to render the README. The style will be clean and readable, with good formatting for code blocks, which is important for our developer audience.

### Step 3: Gesture Handling

- The `RepositoryCard` will be wrapped in a `GestureDetector` to handle swipes.
- **On Horizontal Swipe (Right-to-Left):**
  - The `trackOpenRepository` method from the API service will be called.
  - The `url_launcher` package will be used to open the repository's URL in a new browser tab.
- **On Horizontal Swipe (Left-to-Right):**
  - This will simply be ignored, allowing the user to scroll to the next repository in the list. The primary navigation is vertical scrolling.

## 5. Development & Deployment

- **Development:** Use `flutter run -d chrome` to run and debug the application in a web browser.
- **Web Deployment:** Run `flutter build web` to create a production build. The output in the `build/web` directory can be deployed to any static web host (like Netlify, Vercel, or a simple Nginx server).
- **Mobile Deployment:**
  - **Android:** Run `flutter build apk` or `flutter build appbundle` to create the package for the Google Play Store.
  - **iOS:** Run `flutter build ipa` to create the package for the Apple App Store. (Requires a macOS machine with Xcode).

This plan provides a clear roadmap for developing a robust and user-friendly frontend. We can now proceed with creating the files and writing the code for each of these components.
