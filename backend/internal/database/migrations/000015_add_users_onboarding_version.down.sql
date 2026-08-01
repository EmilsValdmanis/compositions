ALTER TABLE users
DROP CONSTRAINT users_onboarding_version_nonnegative;

ALTER TABLE users
DROP COLUMN onboarding_version;
