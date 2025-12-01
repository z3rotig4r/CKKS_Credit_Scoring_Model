/**
 * React 개발 서버 설정 (setupProxy.js)
 * SharedArrayBuffer 지원을 위한 보안 헤더 추가
 */

module.exports = function(app) {
  // SharedArrayBuffer를 위한 COOP/COEP 헤더 설정
  app.use((req, res, next) => {
    res.setHeader('Cross-Origin-Opener-Policy', 'same-origin');
    res.setHeader('Cross-Origin-Embedder-Policy', 'require-corp');
    next();
  });
};
