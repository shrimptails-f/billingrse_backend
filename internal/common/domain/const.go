package domain

import "time"

// OAuthStateExpirySafetyOffset shortens the effective validity window for safety.
const OAuthStateExpirySafetyOffset = 600 * time.Second // 10分前倒し。
