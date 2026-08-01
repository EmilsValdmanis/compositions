ALTER TABLE users
ADD COLUMN onboarding_version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE users
ALTER COLUMN onboarding_version SET DEFAULT 0;

ALTER TABLE users
ADD CONSTRAINT users_onboarding_version_nonnegative CHECK (onboarding_version >= 0);
