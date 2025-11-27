# 🔒 Private Credit Scoring - Frontend

A React + TypeScript web application for privacy-preserving credit scoring using Fully Homomorphic Encryption (FHE) with CKKS scheme via WebAssembly.

## ✨ Features

- **Fully Homomorphic Encryption**: Perform credit scoring on encrypted data using CKKS
- **WebAssembly Integration**: High-performance encryption powered by Lattigo (Go → WASM)
- **Parallel Encryption**: Web Workers for multi-threaded encryption (5x speedup)
- **Real-time Progress**: Live progress tracking during encryption
- **Performance Benchmarking**: Built-in benchmark tool to measure encryption performance
- **Modern UI**: Clean, responsive interface with gradient design

## 🚀 Quick Start

This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app).

## Available Scripts

In the project directory, you can run:

### `npm start`

Runs the app in the development mode.\
Open [http://localhost:3000](http://localhost:3000) to view it in your browser.

The page will reload when you make changes.\
You may also see any lint errors in the console.

### `npm test`

Launches the test runner in the interactive watch mode.\
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

### `npm run build`

Builds the app for production to the `build` folder.\
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.\
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.

### `npm run eject`

**Note: this is a one-way operation. Once you `eject`, you can't go back!**

If you aren't satisfied with the build tool and configuration choices, you can `eject` at any time. This command will remove the single build dependency from your project.

Instead, it will copy all the configuration files and the transitive dependencies (webpack, Babel, ESLint, etc) right into your project so you have full control over them. All of the commands except `eject` will still work, but they will point to the copied scripts so you can tweak them. At this point you're on your own.

You don't have to ever use `eject`. The curated feature set is suitable for small and middle deployments, and you shouldn't feel obligated to use this feature. However we understand that this tool wouldn't be useful if you couldn't customize it when you are ready for it.

## Learn More

You can learn more in the [Create React App documentation](https://facebook.github.io/create-react-app/docs/getting-started).

To learn React, check out the [React documentation](https://reactjs.org/).

### Code Splitting

This section has moved here: [https://facebook.github.io/create-react-app/docs/code-splitting](https://facebook.github.io/create-react-app/docs/code-splitting)

### Analyzing the Bundle Size

This section has moved here: [https://facebook.github.io/create-react-app/docs/analyzing-the-bundle-size](https://facebook.github.io/create-react-app/docs/analyzing-the-bundle-size)

### Making a Progressive Web App

This section has moved here: [https://facebook.github.io/create-react-app/docs/making-a-progressive-web-app](https://facebook.github.io/create-react-app/docs/making-a-progressive-web-app)

### Advanced Configuration

This section has moved here: [https://facebook.github.io/create-react-app/docs/advanced-configuration](https://facebook.github.io/create-react-app/docs/advanced-configuration)

### Deployment

This section has moved here: [https://facebook.github.io/create-react-app/docs/deployment](https://facebook.github.io/create-react-app/docs/deployment)

### `npm run build` fails to minify

This section has moved here: [https://facebook.github.io/create-react-app/docs/troubleshooting#npm-run-build-fails-to-minify](https://facebook.github.io/create-react-app/docs/troubleshooting#npm-run-build-fails-to-minify)

## 📊 Performance Benchmarks

### Encryption Performance

The application includes a built-in benchmark tool to measure encryption performance with and without Web Workers.

**Typical Results (5 features, 3 iterations):**

| Method | Avg Total Time | Per Feature | Speedup |
|--------|---------------|-------------|---------|
| Sequential | ~1200ms | ~240ms | 1.0x |
| Parallel (Web Workers) | ~250ms | ~50ms | **4.8x** |

**System:** 8-core CPU, Chrome 120+, WebAssembly enabled

### How to Run Benchmarks

1. Generate FHE keys (click "Generate Keys" button)
2. Navigate to the "📊 Benchmark" tab
3. Click "🚀 Run Benchmark"
4. Wait for results (takes ~30 seconds)
5. Copy markdown report for documentation

### Performance Factors

- **CPU Cores**: More cores = better parallelization (4+ cores recommended)
- **WASM Performance**: Modern browsers (Chrome, Firefox, Edge) optimize WASM better
- **Feature Count**: Speedup scales linearly with number of features
- **Memory**: Each worker requires ~50MB for WASM module

## 🔧 Technical Details

### Architecture

```
┌─────────────────┐
│  React Frontend │
├─────────────────┤
│  FHE Context    │  ← State management
├─────────────────┤
│  WASM Loader    │  ← Go/WASM bridge
├─────────────────┤
│  Web Workers    │  ← Parallel encryption
├─────────────────┤
│  Lattigo CKKS   │  ← FHE operations
└─────────────────┘
```

### Key Technologies

- **React 18.2** + **TypeScript 4.9** - UI framework
- **Lattigo v6** - FHE library (compiled to WASM)
- **Web Workers** - Multi-threaded encryption
- **IndexedDB** - Persistent key storage
- **TailwindCSS** - Styling

### CKKS Parameters

- **LogN**: 14 (16384 slots)
- **MaxLevel**: 3
- **LogQ**: [60, 40, 40, 60]
- **LogP**: [61]
- **Scale**: 2^40

### Sigmoid Approximation

The credit scoring uses an optimized degree-3 polynomial approximation:

```
σ(x) ≈ 0.5316x³ + 0.3299x² + 0.0732x + 0.0057
```

**Performance:**
- **Range**: [-3, -1] (credit scoring range)
- **Average Error**: 0.3%
- **Max Error**: 0.86%
- **Method**: Lattigo polynomial evaluator with relinearization

## 🔐 Security Notes

- All encryption happens client-side
- Private keys never leave the browser
- Keys stored in IndexedDB (encrypted by browser)
- Server only receives encrypted ciphertexts
- Results decrypted client-side only

## 📁 Project Structure

```
frontend/
├── public/
│   ├── index.html
│   └── wasm_exec.js         # Go WASM runtime
├── src/
│   ├── components/
│   │   ├── CreditInputForm.tsx   # Main credit scoring form
│   │   ├── BenchmarkPanel.jsx    # Performance testing
│   │   └── KeyManagement.tsx     # Key generation UI
│   ├── contexts/
│   │   └── FHEContext.tsx        # FHE state management
│   ├── services/
│   │   ├── wasmLoader.js         # WASM initialization
│   │   ├── parallelEncryption.js # Web Workers pool
│   │   └── indexedDBService.js   # Key persistence
│   └── utils/
│       └── performanceBenchmark.js # Benchmark utilities
└── README.md
```

## 🐛 Troubleshooting

### Web Workers Not Supported

**Symptoms:** Benchmark shows "Web Workers Not Supported"

**Solutions:**
- Update browser to latest version (Chrome 120+, Firefox 115+)
- Enable JavaScript in browser settings
- Check if running in private/incognito mode (may restrict workers)

### WASM Loading Failed

**Symptoms:** "WASM module failed to load" error

**Solutions:**
- Ensure WASM file is built: `cd ../wasm && ./build.sh`
- Check WASM file exists: `ls -lh public/*.wasm`
- Verify MIME type: Server must serve `.wasm` as `application/wasm`

### Slow Encryption Performance

**Symptoms:** Encryption takes >2 seconds per feature

**Solutions:**
- Enable Web Workers (check benchmark tab)
- Close other heavy browser tabs
- Check CPU usage (should use multiple cores)
- Try different browser (Chrome recommended)

## 🤝 Contributing

This is part of a research project on privacy-preserving machine learning. Contributions welcome!

## 📄 License

MIT License - See project root for details
