# Task 1 Report: Reusable File-Space Scope Migration

## Status

Completed and committed as `2685e60272f3b015105db4b1ee987314ec18f4e3` (`feat(files): add reusable file space scope`).

## Files changed

- `qim-server/model/model.go`
  - Added `File.ScopeType` and `File.ScopeID`.
  - Added the `(scope_type, scope_id, folder_id)` index through GORM field tags.
  - Added `Folder.ScopeType` and `Folder.ScopeID`, while preserving `Folder.UserID` as the creator.
  - Added the `(scope_type, scope_id, parent_id)` folder-tree index through GORM field tags.
  - Kept the pre-existing `File.Source` and `File.SourceID` fields unchanged.
- `qim-server/app/init.go`
  - Added idempotent `MigrateFileSpaces(db *gorm.DB) error`.
  - Invokes the migration after the model AutoMigrate stages in `MigrateDB`.
  - Backfills legacy file and folder rows to `scope_type=user` and `scope_id=user_id`; populated group scopes remain unchanged.
  - Limits backfill strictly to records whose `scope_type` is empty or `NULL`; `scope_type=user, scope_id=0` remains untouched.
- `qim-server/app/file_space_migration_test.go`
  - Covers legacy backfill, preservation of already populated scopes, and the `MigrateDB` integration path.

## TDD evidence

1. Added the migration test before the implementation.
2. Ran:

   ```sh
   cd qim-server && go test ./app -run 'TestMigrateFileSpaces(BackfillsLegacyUserRecords|LeavesExistingScopesUntouched)' -v
   ```

   Initial result: failed to build because `MigrateFileSpaces`, `File.ScopeType`, `File.ScopeID`, `Folder.ScopeType`, and `Folder.ScopeID` did not exist.
3. Added the minimal model and migration implementation, then added the MigrateDB integration regression and verified it fails when the MigrateDB hook is absent.
4. Restored the hook and reran the tests successfully.

## Verification commands and output

```sh
cd qim-server && go test ./app -run 'TestMigrate(FileSpaces|DBBackfillsLegacyFileSpaces)' -v
```

Output: `PASS` — 3 migration tests passed.

```sh
cd qim-server && go test ./app && go test ./service -run 'Test.*(Folder|File)' -v
```

Output: both packages passed (`app` in 7.211s, `service` in 3.515s).

```sh
cd qim-server && go test ./...
```

Output: exit code 0; all discovered server packages completed successfully.

```sh
git diff --check
```

Output: exit code 0; no whitespace errors.

## Concerns

- Existing service tests emit expected SQLite log noise for tables they intentionally do not migrate (for example, `channels` and `folders`); the selected suite still exits successfully. This task does not alter those tests or their setup.
- This task intentionally does not change the existing personal-file service queries. They continue to use `user_id`, preserving their behavior while later scoped-file work adopts the new fields.
- `group_documents` was not modified, so it remains reserved for AI knowledge-base behavior.

## Follow-up review fix

- Corrected `MigrateFileSpaces` to target only `scope_type = '' OR scope_type IS NULL`. This prevents a deliberately stored `user` scope with `scope_id=0` from being rewritten.
- Expanded `TestMigrateFileSpacesBackfillsLegacyUserRecords` to create legacy nullable schemas and exercise both empty-string and `NULL` scope values for files and folders. It also asserts that `user/0` remains unchanged and repeats the migration to verify idempotence.
- Updated the `MigrateDB` integration fixture to explicitly represent an empty legacy scope instead of relying on the current-model default.

### Follow-up TDD and verification evidence

1. The expanded regression was run before the predicate fix and failed as expected: the old migration changed `user/0` to `user/9`.
2. After narrowing the predicate, the following commands completed with exit code 0:

   ```sh
   cd qim-server && go test ./app -run TestMigrateFileSpacesBackfillsLegacyUserRecords -v
   ```

   Output: `PASS` — covers empty and NULL legacy scopes, preserves `user/0`, and verifies a second migration execution.

   ```sh
   cd qim-server && go test ./app -run 'TestMigrate(FileSpaces|DBBackfillsLegacyFileSpaces)' -v
   ```

   Output: `PASS` — 3 migration tests passed.

   ```sh
   cd qim-server && go test ./app
   ```

   Output: `ok github.com/dshmyz/qim/qim-server/app`.
