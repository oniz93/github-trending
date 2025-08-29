// Purpose: Script to select featured repositories using weighted random selection
// Context: Used to maintain a list of featured repositories for display on the homepage

// @ts-expect-error - Bun-specific module
import { Database } from 'bun:sqlite';
import { writeFileSync, existsSync, mkdirSync } from 'fs';
import { dirname } from 'path';

const DB_PATH = 'static/data/db.sqlite';
const OUTPUT_PATH = 'static/data/featured_repos.json';
const NUM_FEATURED_REPOS = 20;

interface Repository {
  id: string;
  owner_id: number;
  name: string;
  full_name: string;
  description: string | null;
  html_url: string;
  language: string | null;
  stargazers_count: number;
  fork: number;
  is_pinned: number;
  default_branch: string | null;
  created_at: string;
  updated_at: string;
  fetched_at: string;
  featured: string | null;
  avatar_url?: string | null;
}

// Extended repository interface with weight property for selection algorithm
interface WeightedRepository extends Repository {
  weight: number;
}

async function main() {
  console.log('Starting featured repositories selection...');
  
  // Open database connection
  const db = new Database(DB_PATH);
  
  // Add featured column if it doesn't exist
  try {
    console.log('Checking if featured column exists...');
    db.query('SELECT featured FROM repositories LIMIT 1').get();
    console.log('Featured column already exists.');
  } catch {
    // Error means the column doesn't exist
    console.log('Adding featured column to repositories table...');
    db.run('ALTER TABLE repositories ADD COLUMN featured TEXT');
    console.log('Featured column added successfully.');
  }
  
  // Add default_branch column if it doesn't exist
  try {
    console.log('Checking if default_branch column exists...');
    db.query('SELECT default_branch FROM repositories LIMIT 1').get();
    console.log('default_branch column already exists.');
  } catch {
    // Error means the column doesn't exist
    console.log('Adding default_branch column to repositories table...');
    db.run('ALTER TABLE repositories ADD COLUMN default_branch TEXT');
    console.log('default_branch column added successfully.');
  }
  
  // Get all pinned repositories with user avatar
  console.log('Fetching pinned repositories...');
  const pinnedRepos = db.query(`
    SELECT r.*, s.avatar_url 
    FROM repositories r
    JOIN stargazers s ON r.owner_id = s.id
    WHERE r.is_pinned = 1
  `).all() as Repository[];
  
  console.log(`Found ${pinnedRepos.length} pinned repositories.`);
  
  if (pinnedRepos.length === 0) {
    console.error('No pinned repositories found. Exiting.');
    db.close();
    process.exit(1);
  }
  
  // Calculate weights based on stars and time since last featured
  const now = new Date().toISOString();
  const reposWithWeights = pinnedRepos.map(repo => {
    // Base weight is the number of stars (minimum 1)
    let weight = Math.max(1, repo.stargazers_count);
    
    // If previously featured, reduce weight based on how recently it was featured
    if (repo.featured) {
      const featuredDate = new Date(repo.featured);
      const daysSinceFeatured = Math.max(1, Math.floor((new Date().getTime() - featuredDate.getTime()) / (1000 * 60 * 60 * 24)));
      
      // Adjust weight: more days since featured = higher weight
      // Less days since featured = lower weight
      weight = weight * (daysSinceFeatured / 30); // Normalize to roughly a month
    } else {
      // If never featured before, give a slight boost
      weight = weight * 1.2;
    }
    
    return {
      ...repo,
      weight
    } as WeightedRepository;
  });
  
  // Perform weighted random selection
  const selectedRepos: WeightedRepository[] = [];
  const totalReposToSelect = Math.min(NUM_FEATURED_REPOS, pinnedRepos.length);
  
  console.log(`Selecting ${totalReposToSelect} repositories...`);
  
  for (let i = 0; i < totalReposToSelect; i++) {
    // Calculate total weight of remaining repositories
    const remainingRepos = reposWithWeights.filter(repo => 
      !selectedRepos.some(selected => selected.id === repo.id)
    );
    
    if (remainingRepos.length === 0) break;
    
    const totalWeight = remainingRepos.reduce((sum, repo) => sum + repo.weight, 0);
    
    // Generate a random value between 0 and totalWeight
    const randomValue = Math.random() * totalWeight;
    
    // Find the repository that corresponds to the random value
    let cumulativeWeight = 0;
    let selectedRepo = remainingRepos[0];
    
    for (const repo of remainingRepos) {
      cumulativeWeight += repo.weight;
      if (randomValue <= cumulativeWeight) {
        selectedRepo = repo;
        break;
      }
    }
    
    // Add the selected repository to our list
    selectedRepos.push(selectedRepo);
  }
  
  // Update the featured column for selected repositories
  console.log('Updating featured timestamp for selected repositories...');
  
  const updateStmt = db.prepare('UPDATE repositories SET featured = ? WHERE id = ?');
  
  for (const repo of selectedRepos) {
    updateStmt.run(now, repo.id);
  }
  
  // Prepare the output data (without the weight property)
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const outputData = selectedRepos.map(({ weight, ...repo }) => repo);
  
  // Ensure the directory exists
  const dir = dirname(OUTPUT_PATH);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  
  // Write to JSON file
  console.log(`Writing ${selectedRepos.length} featured repositories to ${OUTPUT_PATH}...`);
  writeFileSync(OUTPUT_PATH, JSON.stringify(outputData, null, 2));
  
  console.log('Featured repositories selection complete!');
  
  // Close the database connection
  db.close();
}

main().catch(error => {
  console.error('Error running script:', error);
  process.exit(1);
}); 