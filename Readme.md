# SerifHealth Rate Search - Spencer Krigbaum Solution

## Approach 
### For Clarity: I typically do not use AI for my coding, but as this seemed a good oppurtunity to try it out I leveraged Copilot during this assignment.

The general approach I took to this assignment was as follows:
- Manually download the index file
- Manually download the Rate files from within 
    - I specifically chose to manually download them for this as I was not confident that the AI would not repeatedly download those files as I experimented with it.
- I then created a base Go project for the Import, Parsing, and Backend API for the project
- I switched parsing to a python parser contained within the Go Project and hosted it on a sqlLite DB. 
    - I did this due to getting lost in the weeds on the Go SQL Parsing and deciding to take the quicker route with python. In real world scenario I would spend more time looking into the GO Parsing if it was determined Python was not practical for this use case. Sqllite was chosen cause it's easy to use for these kinds of situations.
- After this was inserted into the Sqllite DB I verified and adjusted the data as needed.
    - Originally i did not include the plan data and it's various FK relations to the tables as I built a large portion of the app with only the fidelis-ex file. I went back and added the plan related tables when deciding to import the fidelis-es file as well.
- I then moved to creating the Vue.js frontend
    - I used Copilot heavily here to test it as I have more recently used Vue.js then Go. I had it generate the boilerplate for a vue project, and the App.vue, Seach.Vue, and the .css
    - These came out suprisingly good when first generated but still needed adjustment to the UI and it tended to produce a lot of excess code.
- Iterated on how I wanted to present the data
    - Some solutions I tried: Billing code lookup tab, Plan lookup tab, Provider lookup tab.
    - After spending time on those I decided that an MVP model is best here with those being extensions of the app that could be provided later.
- Settled on this iteration and began vetting the code that the AI had generated. 
    - Used AI to cleanup, look for edge cases, add tests.
    - Overhauled my Go code with the AI, learned a few things including that it is very verbose.


**Assumptions:**
- This did not need to support drop in parsing for the Index File
- That the Index file + in-network files could be added directly to the project
- that including an "Assigment.md" was fine
- The app did not need to be expandable to out-of-network
- Only this Index file would be used, as additonal ones would mix data in the DB without dropping it first.
- Billing Codes, Provider ID's, and Plan ID's are all known by the user (Large assumption, I'm aware.)
- Assumed I could upload the rate files to Github (I was wrong), fixed by adding a download step to the input to grab the "in_network_files"
- I am assuming the "in_network_files" URL is valid for this.


**Limitations:** 
- This does not support drop in Index File parsing
- I think the frontend could use work for clarity in comparing plan vs plan. Perhaps seperate tables where you can take a look at specific procedure's side by side between plans.
- Doing some analysis on the data could be helpful as well with a way to display "X plan is Y% higher on average per procedure" **This would be my next step for the application**
- While I added pagination for the data to the frontend it does not limit or try to be performant during data ingress. Some of the CMS files can be hundred of gig's (From what I researched) in size and that would need to be adjusted for here.
- From what I saw another limitation that is likely not our concern but would be for our clients is that the CMS data only shows items/services that have over 11 historic claims (used to be 20).
- I would add a fallback case if the "in_network_files" url is invalid, such as retriving a new index file.

**Overall take away with Copilot:** 
I would likely use it on future projects for frontend boilerplate and placeholders, I would use it for testing and edge case checking. I would likely not use it for backend as it just makes it to verbose generally and could be done much cleaner.

----------------------------------------------------------------------------------------------------------------------------------



Install Python 3, Go 1.27.1 or newer, and Node.js 18 or newer. Run all commands from the project folder.

Create the database once:

```powershell
# Windows
py -3 import_sqlite.py
```

```bash
# Linux
python3 import_sqlite.py
```

Start the backend in one terminal:

```bash
go run .
```

Start the frontend in a second terminal:

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173. The importer creates `serifhealth.db` from both network files and the parent index.