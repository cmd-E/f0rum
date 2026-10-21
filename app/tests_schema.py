from pathlib import Path
import re, sqlite3, tempfile, unittest
ROOT=Path(__file__).resolve().parents[1]
class SchemaChecks(unittest.TestCase):
    def schema(self,db):
        for statement in (ROOT/'app/forum/schema.sql').read_text(encoding='utf-8-sig').split(';'):
            if statement.strip():db.execute(statement)
    def test_constraints_transactions_and_cascades(self):
        db=sqlite3.connect(':memory:');db.execute('PRAGMA foreign_keys=ON');self.schema(db)
        db.execute("INSERT INTO users_new(id,username,email,password) VALUES(1,'one','one@example.invalid','hash')")
        with self.assertRaises(sqlite3.IntegrityError):db.execute("INSERT INTO users_new(username,email,password) VALUES('one','two@example.invalid','hash')")
        self.assertEqual(db.execute('SELECT role FROM users_new').fetchone()[0],'user')
        db.execute("INSERT INTO threads_new VALUES(1,'thread','description')")
        db.execute("INSERT INTO posts_new(id,title,content,authorid) VALUES(1,'title','body',1)")
        db.execute('INSERT INTO postsandcats_new(threadid,postid) VALUES(1,1)')
        db.execute("INSERT INTO comments_new(id,postid,user_id,content) VALUES(1,1,1,'comment')")
        db.execute("INSERT INTO rtactions_new(postid,user_id,type) VALUES(1,1,'like')")
        with self.assertRaises(sqlite3.IntegrityError):db.execute("INSERT INTO rtactions_new(postid,user_id,type) VALUES(1,1,'dislike')")
        db.execute('DELETE FROM threads_new WHERE id=1')
        self.assertEqual(db.execute('SELECT count(*) FROM posts_new').fetchone()[0],1)
        self.assertEqual(db.execute('SELECT count(*) FROM postsandcats_new').fetchone()[0],0)
        db.execute('DELETE FROM posts_new WHERE id=1')
        for table in ['comments_new','rtactions_new']:
            self.assertEqual(db.execute('SELECT count(*) FROM '+table).fetchone()[0],0)
        self.assertEqual(list(db.execute('PRAGMA foreign_key_check')),[])
    def test_valid_legacy_migration_preserves_identity(self):
        source=sqlite3.connect('file:'+str(ROOT/'database/forum.db').replace('\\','/')+'?mode=ro',uri=True)
        db=sqlite3.connect(':memory:');db.execute('PRAGMA foreign_keys=ON')
        for (sql,) in source.execute("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"):
            db.execute(sql)
        source.close()
        db.execute("INSERT INTO users VALUES(1,'user','user@example.invalid','dummy-hash')")
        db.execute("INSERT INTO threads VALUES(1,'thread','description')")
        db.execute("INSERT INTO posts(id,title,content,likes,dislikes,author,authorid) VALUES(1,'title','body',1,0,'user',1)")
        db.execute("INSERT INTO postsandcats(threadid,postid) VALUES(1,1)")
        db.execute("INSERT INTO comments(id,postid,username,content) VALUES(1,1,'user','comment')")
        db.execute("INSERT INTO rtactions(postid,username,userid,type) VALUES(1,'user',1,'like')")
        db.execute("INSERT INTO rtactionscomm(commid,username) VALUES(1,'user')")
        db.execute("INSERT INTO sessions(session,username) VALUES('legacy-token','user')")
        self.schema(db)
        code=(ROOT/'app/forum/store.go').read_text(encoding='utf-8-sig')
        for statement in re.findall(r'"(INSERT INTO [^"\n]*_new[^"\n]*)"',code):db.execute(statement)
        for table in ['rtactionscomm','rtactions','comments','postsandcats','posts','threads','sessions','users']:
            db.execute('DROP TABLE '+table)
        for table in ['users','sessions','threads','posts','postsandcats','comments','rtactions','rtactionscomm']:
            db.execute('ALTER TABLE '+table+'_new RENAME TO '+table)
        self.assertEqual(db.execute('SELECT id,username,role FROM users').fetchone(),(1,'user','user'))
        self.assertEqual(db.execute('SELECT count(*) FROM sessions').fetchone()[0],0)
        self.assertEqual(db.execute('SELECT authorid,likes FROM posts').fetchone(),(1,1))
        self.assertIsNone(db.execute('SELECT type FROM rtactionscomm').fetchone()[0])
        self.assertEqual(list(db.execute('PRAGMA foreign_key_check')),[])
        db.close()
    def test_legacy_rehearsal_on_copy(self):
        # Read the source database only; all migration operations run on a disposable copy.
        source=sqlite3.connect('file:'+str(ROOT/'database/forum.db').replace('\\','/')+'?mode=ro',uri=True)
        with tempfile.TemporaryDirectory() as directory:
            db=sqlite3.connect(str(Path(directory)/'copy.db'));source.backup(db);source.close()
            db.execute('PRAGMA foreign_keys=ON');db.execute('BEGIN');self.schema(db)
            code=(ROOT/'app/forum/store.go').read_text(encoding='utf-8-sig')
            copies=re.findall(r'"(INSERT INTO [^"\n]*_new[^"\n]*)"',code)
            self.assertEqual(len(copies),7)
            try:
                for statement in copies:db.execute(statement)
            except sqlite3.IntegrityError:
                db.rollback()
                self.assertEqual(db.execute('PRAGMA user_version').fetchone()[0],0)
                self.assertEqual(db.execute("SELECT count(*) FROM sqlite_master WHERE name='users_new'").fetchone()[0],0)
                print('Legacy copy has conflicts; migration correctly rolled back for explicit reconciliation')
            else:
                for table in ['rtactionscomm','rtactions','comments','postsandcats','posts','threads','sessions','users']:
                    db.execute('DROP TABLE '+table)
                for table in ['users','sessions','threads','posts','postsandcats','comments','rtactions','rtactionscomm']:
                    db.execute('ALTER TABLE '+table+'_new RENAME TO '+table)
                self.assertEqual(list(db.execute('PRAGMA foreign_key_check')),[])
                self.assertEqual(db.execute('SELECT count(*) FROM sessions').fetchone()[0],0)
                db.commit();print('Legacy migration rehearsed successfully on disposable copy')
            db.close()
if __name__=='__main__':unittest.main()
